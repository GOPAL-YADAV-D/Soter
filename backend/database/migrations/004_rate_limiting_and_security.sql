-- Database schema extensions for rate limiting, quotas, and security features
-- Run after existing migrations

-- Rate limiting configuration per user/organization
CREATE TABLE IF NOT EXISTS rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'organization')),
    entity_id UUID NOT NULL,
    requests_per_second INTEGER NOT NULL DEFAULT 2,
    burst_capacity INTEGER NOT NULL DEFAULT 5,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id)
);

-- Create indexes for rate limits
CREATE INDEX IF NOT EXISTS idx_rate_limits_entity ON rate_limits(entity_type, entity_id);

-- Quota events for monitoring and alerts
CREATE TABLE IF NOT EXISTS quota_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('quota_exceeded', 'quota_warning', 'quota_reset')),
    quota_bytes BIGINT NOT NULL,
    used_bytes BIGINT NOT NULL,
    file_size_bytes BIGINT,
    usage_percent DECIMAL(5,2) NOT NULL,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for quota events
CREATE INDEX IF NOT EXISTS idx_quota_events_user_id ON quota_events(user_id);
CREATE INDEX IF NOT EXISTS idx_quota_events_org_id ON quota_events(organization_id);
CREATE INDEX IF NOT EXISTS idx_quota_events_type ON quota_events(event_type);
CREATE INDEX IF NOT EXISTS idx_quota_events_created_at ON quota_events(created_at);

-- Audit logs for security and compliance
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    details JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failure', 'error')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for audit logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(organization_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address);

-- File validation results for virus scanning and content analysis
CREATE TABLE IF NOT EXISTS file_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    validation_type VARCHAR(50) NOT NULL CHECK (validation_type IN ('virus_scan', 'content_analysis', 'mime_check')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'clean', 'infected', 'suspicious', 'error')),
    engine VARCHAR(50),
    engine_version VARCHAR(50),
    scan_result JSONB,
    details TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for file validations
CREATE INDEX IF NOT EXISTS idx_file_validations_file_id ON file_validations(file_id);
CREATE INDEX IF NOT EXISTS idx_file_validations_type ON file_validations(validation_type);
CREATE INDEX IF NOT EXISTS idx_file_validations_status ON file_validations(status);
CREATE INDEX IF NOT EXISTS idx_file_validations_created_at ON file_validations(created_at);

-- User sessions for enhanced security tracking
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for user sessions
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(is_active);

-- Storage usage history for analytics and monitoring
CREATE TABLE IF NOT EXISTS storage_usage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    total_files INTEGER NOT NULL DEFAULT 0,
    unique_files INTEGER NOT NULL DEFAULT 0,
    total_size_bytes BIGINT NOT NULL DEFAULT 0,
    actual_storage_bytes BIGINT NOT NULL DEFAULT 0,
    savings_bytes BIGINT NOT NULL DEFAULT 0,
    savings_percentage DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for storage usage history
CREATE INDEX IF NOT EXISTS idx_storage_usage_history_user_id ON storage_usage_history(user_id);
CREATE INDEX IF NOT EXISTS idx_storage_usage_history_org_id ON storage_usage_history(organization_id);
CREATE INDEX IF NOT EXISTS idx_storage_usage_history_recorded_at ON storage_usage_history(recorded_at);

-- Add some useful functions for quota management

-- Function to get current storage usage for a user
CREATE OR REPLACE FUNCTION get_user_storage_usage(p_user_id UUID)
RETURNS TABLE(
    total_files INTEGER,
    unique_files INTEGER,
    total_size_bytes BIGINT,
    actual_storage_bytes BIGINT,
    savings_bytes BIGINT,
    savings_percentage DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(uf.id)::INTEGER as total_files,
        COUNT(DISTINCT f.content_hash)::INTEGER as unique_files,
        COALESCE(SUM(f.file_size), 0) as total_size_bytes,
        COALESCE(SUM(DISTINCT f.file_size), 0) as actual_storage_bytes,
        COALESCE(SUM(f.file_size) - SUM(DISTINCT f.file_size), 0) as savings_bytes,
        CASE 
            WHEN SUM(f.file_size) > 0 THEN 
                ROUND(((SUM(f.file_size) - SUM(DISTINCT f.file_size))::DECIMAL / SUM(f.file_size) * 100), 2)
            ELSE 0 
        END as savings_percentage
    FROM user_files uf
    JOIN files f ON uf.file_id = f.id
    WHERE uf.user_id = p_user_id 
    AND uf.is_deleted = false;
END;
$$ LANGUAGE plpgsql;

-- Function to record quota events
CREATE OR REPLACE FUNCTION record_quota_event(
    p_user_id UUID,
    p_organization_id UUID,
    p_event_type VARCHAR,
    p_quota_bytes BIGINT,
    p_used_bytes BIGINT,
    p_file_size_bytes BIGINT DEFAULT NULL,
    p_details JSONB DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    event_id UUID;
    usage_percent DECIMAL;
BEGIN
    -- Calculate usage percentage
    usage_percent := CASE 
        WHEN p_quota_bytes > 0 THEN ROUND((p_used_bytes::DECIMAL / p_quota_bytes * 100), 2)
        ELSE 0 
    END;
    
    -- Insert quota event
    INSERT INTO quota_events (
        user_id, organization_id, event_type, quota_bytes, used_bytes, 
        file_size_bytes, usage_percent, details
    ) VALUES (
        p_user_id, p_organization_id, p_event_type, p_quota_bytes, p_used_bytes,
        p_file_size_bytes, usage_percent, p_details
    ) RETURNING id INTO event_id;
    
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update storage stats when files are added/removed
CREATE OR REPLACE FUNCTION update_storage_stats() RETURNS TRIGGER AS $$
BEGIN
    -- This would be implemented to automatically update user_storage_stats
    -- when user_files table changes
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create trigger (commented out for now - implement when ready)
-- DROP TRIGGER IF EXISTS tr_update_storage_stats ON user_files;
-- CREATE TRIGGER tr_update_storage_stats
--     AFTER INSERT OR UPDATE OR DELETE ON user_files
--     FOR EACH ROW EXECUTE FUNCTION update_storage_stats();

-- Add comments to tables for documentation
COMMENT ON TABLE rate_limits IS 'Custom rate limiting configuration per user or organization';
COMMENT ON TABLE quota_events IS 'Events related to storage quota usage for monitoring and alerts';
COMMENT ON TABLE audit_logs IS 'Comprehensive audit trail of all system actions for security and compliance';
COMMENT ON TABLE file_validations IS 'Results of file validation including virus scanning and content analysis';
COMMENT ON TABLE user_sessions IS 'Active user sessions for enhanced security tracking';
COMMENT ON TABLE storage_usage_history IS 'Historical storage usage data for analytics and monitoring';