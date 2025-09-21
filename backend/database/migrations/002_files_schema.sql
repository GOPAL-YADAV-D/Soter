-- Database schema for file management with deduplication
-- Run after 001_initial_schema.sql

-- Files table - stores unique file content by hash
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA-256 hash
    filename VARCHAR(255) NOT NULL,
    original_mime_type VARCHAR(100) NOT NULL,
    detected_mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_path VARCHAR(500) NOT NULL, -- Azure Blob Storage path
    upload_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User files table - maps users to files (many-to-many for deduplication)
CREATE TABLE IF NOT EXISTS user_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    user_filename VARCHAR(255) NOT NULL, -- User's custom filename
    upload_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,
    folder_path VARCHAR(500) DEFAULT '/', -- For organizing files
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, user_filename, folder_path)
);

-- Storage savings tracking per user
CREATE TABLE IF NOT EXISTS user_storage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_files INTEGER DEFAULT 0,
    unique_files INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    actual_storage_bytes BIGINT DEFAULT 0,
    savings_bytes BIGINT DEFAULT 0,
    savings_percentage DECIMAL(5,2) DEFAULT 0.00,
    last_calculated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- File upload sessions for tracking progress
CREATE TABLE IF NOT EXISTS upload_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    total_files INTEGER NOT NULL,
    completed_files INTEGER DEFAULT 0,
    failed_files INTEGER DEFAULT 0,
    total_bytes BIGINT NOT NULL,
    uploaded_bytes BIGINT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending', -- pending, in_progress, completed, failed
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- File chunks for resumable uploads (optional for large files)
CREATE TABLE IF NOT EXISTS file_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_session_id UUID NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    file_hash VARCHAR(64) NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_size INTEGER NOT NULL,
    chunk_hash VARCHAR(64) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending, uploaded, verified
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(upload_session_id, file_hash, chunk_index)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_files_content_hash ON files(content_hash);
CREATE INDEX IF NOT EXISTS idx_files_upload_date ON files(upload_date);
CREATE INDEX IF NOT EXISTS idx_user_files_user_id ON user_files(user_id);
CREATE INDEX IF NOT EXISTS idx_user_files_file_id ON user_files(file_id);
CREATE INDEX IF NOT EXISTS idx_user_files_folder_path ON user_files(folder_path);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_user_id ON upload_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_token ON upload_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_file_chunks_session_id ON file_chunks(upload_session_id);

-- Functions to automatically update storage stats
CREATE OR REPLACE FUNCTION update_user_storage_stats(p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    INSERT INTO user_storage_stats (
        user_id,
        total_files,
        unique_files,
        total_size_bytes,
        actual_storage_bytes,
        savings_bytes,
        savings_percentage,
        last_calculated
    )
    SELECT
        p_user_id,
        COUNT(uf.id) as total_files,
        COUNT(DISTINCT uf.file_id) as unique_files,
        COALESCE(SUM(f.file_size), 0) as total_size_bytes,
        COALESCE(SUM(DISTINCT f.file_size), 0) as actual_storage_bytes,
        COALESCE(SUM(f.file_size) - SUM(DISTINCT f.file_size), 0) as savings_bytes,
        CASE 
            WHEN SUM(f.file_size) > 0 THEN 
                ROUND(((SUM(f.file_size) - SUM(DISTINCT f.file_size))::DECIMAL / SUM(f.file_size)) * 100, 2)
            ELSE 0.00
        END as savings_percentage,
        CURRENT_TIMESTAMP
    FROM user_files uf
    JOIN files f ON uf.file_id = f.id
    WHERE uf.user_id = p_user_id AND uf.is_deleted = FALSE
    ON CONFLICT (user_id) DO UPDATE SET
        total_files = EXCLUDED.total_files,
        unique_files = EXCLUDED.unique_files,
        total_size_bytes = EXCLUDED.total_size_bytes,
        actual_storage_bytes = EXCLUDED.actual_storage_bytes,
        savings_bytes = EXCLUDED.savings_bytes,
        savings_percentage = EXCLUDED.savings_percentage,
        last_calculated = EXCLUDED.last_calculated,
        updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update storage stats when user_files changes
CREATE OR REPLACE FUNCTION trigger_update_storage_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM update_user_storage_stats(OLD.user_id);
        RETURN OLD;
    ELSE
        PERFORM update_user_storage_stats(NEW.user_id);
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_files_storage_stats_trigger
    AFTER INSERT OR UPDATE OR DELETE ON user_files
    FOR EACH ROW EXECUTE FUNCTION trigger_update_storage_stats();