-- Enhanced schema for groups, permissions, and storage allocation

-- Add allocated_space_mb to organizations table
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS allocated_space_mb INTEGER DEFAULT 100;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS used_space_mb INTEGER DEFAULT 0;

-- Create groups table (Linux-style access control)
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL, -- e.g., 'root', 'admin', 'users', 'guests'
    description TEXT,
    permissions INTEGER DEFAULT 755, -- Linux-style permissions (rwxrwxrwx)
    is_system_group BOOLEAN DEFAULT false, -- System groups like 'root', 'admin'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, name)
);

-- Create user_groups junction table (many-to-many)
CREATE TABLE IF NOT EXISTS user_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by UUID REFERENCES users(id),
    UNIQUE(user_id, group_id)
);

-- Create file_group_permissions table (file-level access control)
CREATE TABLE IF NOT EXISTS file_group_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    permissions INTEGER DEFAULT 644, -- Linux-style permissions for this group
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(file_id, group_id)
);

-- Update files table to include owner and primary group
ALTER TABLE files ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES users(id);
ALTER TABLE files ADD COLUMN IF NOT EXISTS primary_group_id UUID REFERENCES groups(id);
ALTER TABLE files ADD COLUMN IF NOT EXISTS file_permissions INTEGER DEFAULT 644; -- rwxrwxrwx format

-- Update user_files table to include additional metadata
ALTER TABLE user_files ADD COLUMN IF NOT EXISTS download_count INTEGER DEFAULT 0;
ALTER TABLE user_files ADD COLUMN IF NOT EXISTS last_accessed TIMESTAMP WITH TIME ZONE;
ALTER TABLE user_files ADD COLUMN IF NOT EXISTS access_permissions INTEGER DEFAULT 644;

-- Create organization_storage_stats table for detailed tracking
CREATE TABLE IF NOT EXISTS organization_storage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    allocated_space_mb INTEGER NOT NULL,
    used_space_mb INTEGER DEFAULT 0,
    file_count INTEGER DEFAULT 0,
    unique_file_count INTEGER DEFAULT 0,
    deduplication_savings_mb INTEGER DEFAULT 0,
    last_calculated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_groups_organization ON groups(organization_id);
CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(organization_id, name);
CREATE INDEX IF NOT EXISTS idx_user_groups_user ON user_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_user_groups_group ON user_groups(group_id);
CREATE INDEX IF NOT EXISTS idx_file_group_permissions_file ON file_group_permissions(file_id);
CREATE INDEX IF NOT EXISTS idx_file_group_permissions_group ON file_group_permissions(group_id);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files(owner_id);
CREATE INDEX IF NOT EXISTS idx_files_primary_group ON files(primary_group_id);

-- Create function to update organization storage statistics
CREATE OR REPLACE FUNCTION update_organization_storage_stats()
RETURNS TRIGGER AS $$
DECLARE
    org_id UUID;
    total_logical_size BIGINT := 0;
    total_physical_size BIGINT := 0;
    total_files INTEGER := 0;
    unique_files INTEGER := 0;
BEGIN
    -- Get organization ID from the file
    IF TG_OP = 'INSERT' THEN
        SELECT uf.user_id INTO org_id 
        FROM user_files uf 
        JOIN users u ON uf.user_id = u.id 
        WHERE uf.id = NEW.id;
        
        SELECT u.organization_id INTO org_id
        FROM users u
        WHERE u.id = (SELECT uf.user_id FROM user_files uf WHERE uf.id = NEW.id);
    ELSIF TG_OP = 'DELETE' THEN
        SELECT u.organization_id INTO org_id
        FROM users u
        WHERE u.id = (SELECT uf.user_id FROM user_files uf WHERE uf.id = OLD.id);
    END IF;

    -- Calculate storage statistics for the organization
    SELECT 
        COALESCE(SUM(f.file_size), 0) / (1024 * 1024), -- Convert to MB
        COALESCE(SUM(DISTINCT f.file_size), 0) / (1024 * 1024), -- Unique files only
        COUNT(*),
        COUNT(DISTINCT f.content_hash)
    INTO total_logical_size, total_physical_size, total_files, unique_files
    FROM user_files uf
    JOIN files f ON uf.file_id = f.id
    JOIN users u ON uf.user_id = u.id
    WHERE u.organization_id = org_id AND uf.is_deleted = false;

    -- Update organization storage stats
    INSERT INTO organization_storage_stats (
        organization_id, allocated_space_mb, used_space_mb, 
        file_count, unique_file_count, deduplication_savings_mb
    ) VALUES (
        org_id, 
        (SELECT allocated_space_mb FROM organizations WHERE id = org_id),
        total_physical_size,
        total_files,
        unique_files,
        (total_logical_size - total_physical_size)
    )
    ON CONFLICT (organization_id) 
    DO UPDATE SET
        used_space_mb = total_physical_size,
        file_count = total_files,
        unique_file_count = unique_files,
        deduplication_savings_mb = (total_logical_size - total_physical_size),
        last_calculated = CURRENT_TIMESTAMP;

    -- Update organization used_space_mb
    UPDATE organizations 
    SET used_space_mb = total_physical_size
    WHERE id = org_id;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for storage stats updates
DROP TRIGGER IF EXISTS update_org_storage_on_user_file_insert ON user_files;
CREATE TRIGGER update_org_storage_on_user_file_insert
    AFTER INSERT ON user_files
    FOR EACH ROW EXECUTE FUNCTION update_organization_storage_stats();

DROP TRIGGER IF EXISTS update_org_storage_on_user_file_delete ON user_files;
CREATE TRIGGER update_org_storage_on_user_file_delete
    AFTER DELETE ON user_files
    FOR EACH ROW EXECUTE FUNCTION update_organization_storage_stats();

-- Function to create default groups for new organizations
CREATE OR REPLACE FUNCTION create_default_groups_for_organization()
RETURNS TRIGGER AS $$
BEGIN
    -- Create root/admin group (full permissions)
    INSERT INTO groups (organization_id, name, description, permissions, is_system_group)
    VALUES (NEW.id, 'admin', 'Administrator group with full permissions', 777, true);
    
    -- Create users group (read/write permissions)
    INSERT INTO groups (organization_id, name, description, permissions, is_system_group)
    VALUES (NEW.id, 'users', 'Standard users group', 664, true);
    
    -- Create guests group (read-only permissions)
    INSERT INTO groups (organization_id, name, description, permissions, is_system_group)
    VALUES (NEW.id, 'guests', 'Guest users with read-only access', 444, true);
    
    -- Initialize organization storage stats
    INSERT INTO organization_storage_stats (organization_id, allocated_space_mb)
    VALUES (NEW.id, NEW.allocated_space_mb);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for default groups creation
DROP TRIGGER IF EXISTS create_default_groups_trigger ON organizations;
CREATE TRIGGER create_default_groups_trigger
    AFTER INSERT ON organizations
    FOR EACH ROW EXECUTE FUNCTION create_default_groups_for_organization();

-- Function to assign creator to admin group
CREATE OR REPLACE FUNCTION assign_creator_to_admin_group()
RETURNS TRIGGER AS $$
DECLARE
    admin_group_id UUID;
BEGIN
    -- Get the admin group for this organization
    SELECT id INTO admin_group_id 
    FROM groups 
    WHERE organization_id = NEW.organization_id AND name = 'admin' AND is_system_group = true;
    
    -- Assign user to admin group if they have admin role
    IF NEW.role = 'admin' AND admin_group_id IS NOT NULL THEN
        INSERT INTO user_groups (user_id, group_id, assigned_by)
        VALUES (NEW.user_id, admin_group_id, NEW.user_id)
        ON CONFLICT (user_id, group_id) DO NOTHING;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for admin group assignment
DROP TRIGGER IF EXISTS assign_admin_group_trigger ON user_organizations;
CREATE TRIGGER assign_admin_group_trigger
    AFTER INSERT ON user_organizations
    FOR EACH ROW EXECUTE FUNCTION assign_creator_to_admin_group();

-- Create view for user permissions summary
CREATE OR REPLACE VIEW user_permissions_summary AS
SELECT 
    u.id as user_id,
    u.username,
    u.email,
    o.id as organization_id,
    o.name as organization_name,
    uo.role as organization_role,
    g.id as group_id,
    g.name as group_name,
    g.permissions as group_permissions,
    g.is_system_group
FROM users u
JOIN user_organizations uo ON u.id = uo.user_id
JOIN organizations o ON uo.organization_id = o.id
LEFT JOIN user_groups ug ON u.id = ug.user_id
LEFT JOIN groups g ON ug.group_id = g.id AND g.organization_id = o.id;

-- Create view for file access permissions
CREATE OR REPLACE VIEW file_access_permissions AS
SELECT 
    f.id as file_id,
    f.user_filename,
    f.folder_path,
    files.file_size,
    files.content_hash,
    u.id as user_id,
    u.username,
    o.id as organization_id,
    o.name as organization_name,
    g.id as group_id,
    g.name as group_name,
    COALESCE(fgp.permissions, g.permissions, files.file_permissions) as effective_permissions
FROM user_files f
JOIN files ON f.file_id = files.id
JOIN users u ON f.user_id = u.id
JOIN user_organizations uo ON u.id = uo.user_id
JOIN organizations o ON uo.organization_id = o.id
LEFT JOIN user_groups ug ON u.id = ug.user_id
LEFT JOIN groups g ON ug.group_id = g.id AND g.organization_id = o.id
LEFT JOIN file_group_permissions fgp ON files.id = fgp.file_id AND g.id = fgp.group_id
WHERE f.is_deleted = false;