-- Initialize database schema for Secure File Vault System

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMP WITH TIME ZONE,
    storage_quota_mb INTEGER DEFAULT 10
);

-- Create organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_by_user_id UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

-- Create user_organizations junction table (many-to-many)
CREATE TABLE user_organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member', -- admin, editor, viewer, member
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, organization_id)
);

-- Create blob_metadata table for deduplication
CREATE TABLE blob_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sha256_hash VARCHAR(64) UNIQUE NOT NULL,
    size_bytes BIGINT NOT NULL,
    content_type VARCHAR(255),
    reference_count INTEGER DEFAULT 0,
    azure_blob_path TEXT, -- Path in Azure Blob Storage
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create files table
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256_hash VARCHAR(64) NOT NULL REFERENCES blob_metadata(sha256_hash),
    uploaded_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    download_count INTEGER DEFAULT 0,
    is_public BOOLEAN DEFAULT FALSE,
    share_token VARCHAR(255) UNIQUE,
    share_expires_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID REFERENCES organizations(id),
    folder_path TEXT DEFAULT '/', -- For future folder hierarchy
    tags TEXT[], -- Array of tags for search
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create file permissions table (Linux-style permissions)
CREATE TABLE file_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    owner_permissions INTEGER DEFAULT 7, -- rwx for owner (4+2+1)
    group_permissions INTEGER DEFAULT 5, -- r-x for group (4+0+1)
    other_permissions INTEGER DEFAULT 0, -- --- for others
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create audit_logs table for compliance
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(50) NOT NULL, -- upload, download, delete, share, etc.
    resource_type VARCHAR(50) NOT NULL, -- file, user, organization
    resource_id UUID,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create storage_statistics table for tracking usage
CREATE TABLE storage_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    logical_storage_bytes BIGINT DEFAULT 0,
    physical_storage_bytes BIGINT DEFAULT 0,
    file_count INTEGER DEFAULT 0,
    last_calculated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, organization_id)
);

-- Create indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_active ON users(is_active);

CREATE INDEX idx_files_uploaded_by ON files(uploaded_by);
CREATE INDEX idx_files_organization ON files(organization_id);
CREATE INDEX idx_files_sha256 ON files(sha256_hash);
CREATE INDEX idx_files_public ON files(is_public);
CREATE INDEX idx_files_deleted ON files(is_deleted);
CREATE INDEX idx_files_upload_date ON files(uploaded_at);
CREATE INDEX idx_files_filename_gin ON files USING gin(filename gin_trgm_ops);
CREATE INDEX idx_files_tags_gin ON files USING gin(tags);

CREATE INDEX idx_blob_metadata_hash ON blob_metadata(sha256_hash);
CREATE INDEX idx_blob_metadata_ref_count ON blob_metadata(reference_count);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

CREATE INDEX idx_user_orgs_user ON user_organizations(user_id);
CREATE INDEX idx_user_orgs_org ON user_organizations(organization_id);

-- Create triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_file_permissions_updated_at BEFORE UPDATE ON file_permissions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to update blob reference count
CREATE OR REPLACE FUNCTION update_blob_reference_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE blob_metadata 
        SET reference_count = reference_count + 1
        WHERE sha256_hash = NEW.sha256_hash;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE blob_metadata 
        SET reference_count = reference_count - 1
        WHERE sha256_hash = OLD.sha256_hash;
        -- TODO: Add garbage collection for blobs with reference_count = 0
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_blob_ref_count_on_file_insert 
    AFTER INSERT ON files
    FOR EACH ROW EXECUTE FUNCTION update_blob_reference_count();

CREATE TRIGGER update_blob_ref_count_on_file_delete 
    AFTER DELETE ON files
    FOR EACH ROW EXECUTE FUNCTION update_blob_reference_count();

-- Insert default admin user (password: admin123 - change in production!)
INSERT INTO users (username, email, password_hash, storage_quota_mb) VALUES 
('admin', 'admin@soter.local', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 1000);

-- Insert default organization
INSERT INTO organizations (name, description) VALUES 
('Default Organization', 'Default organization for all users');

-- Add admin to default organization as admin
INSERT INTO user_organizations (user_id, organization_id, role) 
SELECT u.id, o.id, 'admin' 
FROM users u, organizations o 
WHERE u.username = 'admin' AND o.name = 'Default Organization';