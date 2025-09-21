-- Migration 005: Audit Logs Table
-- Create audit_logs table for comprehensive audit logging

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    details JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action ON audit_logs(user_id, action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_time_user ON audit_logs(created_at, user_id);

-- Create GIN index for JSONB details column for efficient JSON queries
CREATE INDEX IF NOT EXISTS idx_audit_logs_details ON audit_logs USING GIN(details);

-- Add comments for documentation
COMMENT ON TABLE audit_logs IS 'Comprehensive audit log for all system activities';
COMMENT ON COLUMN audit_logs.id IS 'Unique identifier for the audit log entry';
COMMENT ON COLUMN audit_logs.user_id IS 'ID of the user who performed the action (nullable for system actions)';
COMMENT ON COLUMN audit_logs.organization_id IS 'ID of the organization context for the action';
COMMENT ON COLUMN audit_logs.action IS 'Type of action performed (e.g., file_upload, user_login)';
COMMENT ON COLUMN audit_logs.resource_type IS 'Type of resource affected (e.g., file, user, permission)';
COMMENT ON COLUMN audit_logs.resource_id IS 'ID of the specific resource affected';
COMMENT ON COLUMN audit_logs.ip_address IS 'IP address from which the action was performed';
COMMENT ON COLUMN audit_logs.user_agent IS 'User agent string from the client';
COMMENT ON COLUMN audit_logs.request_id IS 'Unique identifier for the HTTP request';
COMMENT ON COLUMN audit_logs.details IS 'Additional context and metadata for the action';
COMMENT ON COLUMN audit_logs.status IS 'Status of the action (success, failure, error)';
COMMENT ON COLUMN audit_logs.created_at IS 'Timestamp when the action was performed';

-- Create a view for recent security events
CREATE OR REPLACE VIEW recent_security_events AS
SELECT 
    id,
    user_id,
    action,
    resource_type,
    resource_id,
    ip_address,
    details,
    status,
    created_at
FROM audit_logs 
WHERE 
    (action IN ('user_login', 'user_logout', 'permission_change', 'security_violation', 'rate_limit_exceeded')
     OR resource_type = 'security')
    AND created_at >= NOW() - INTERVAL '7 days'
ORDER BY created_at DESC;

-- Create a view for failed actions requiring attention
CREATE OR REPLACE VIEW failed_actions AS
SELECT 
    id,
    user_id,
    action,
    resource_type,
    resource_id,
    ip_address,
    user_agent,
    details,
    created_at
FROM audit_logs 
WHERE 
    status IN ('failure', 'error')
    AND created_at >= NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- Create a function to clean up old audit logs (for maintenance)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(retention_days INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_logs 
    WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Log the cleanup action
    INSERT INTO audit_logs (action, resource_type, details, status)
    VALUES (
        'audit_cleanup',
        'system',
        jsonb_build_object(
            'deleted_count', deleted_count,
            'retention_days', retention_days,
            'cleanup_date', NOW()
        ),
        'success'
    );
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a function to get audit statistics
CREATE OR REPLACE FUNCTION get_audit_statistics(days INTEGER DEFAULT 7)
RETURNS TABLE(
    action VARCHAR,
    resource_type VARCHAR,
    success_count BIGINT,
    failure_count BIGINT,
    total_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.action,
        a.resource_type,
        COUNT(*) FILTER (WHERE a.status = 'success') as success_count,
        COUNT(*) FILTER (WHERE a.status IN ('failure', 'error')) as failure_count,
        COUNT(*) as total_count
    FROM audit_logs a
    WHERE a.created_at >= NOW() - (days || ' days')::INTERVAL
    GROUP BY a.action, a.resource_type
    ORDER BY total_count DESC;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to automatically partition audit logs by month (for large deployments)
-- This is optional but recommended for high-volume systems
CREATE OR REPLACE FUNCTION create_monthly_audit_partition()
RETURNS TRIGGER AS $$
DECLARE
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- Calculate partition name and date range
    start_date := DATE_TRUNC('month', NEW.created_at)::DATE;
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'audit_logs_' || TO_CHAR(start_date, 'YYYY_MM');
    
    -- Create partition if it doesn't exist
    BEGIN
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs 
                       FOR VALUES FROM (%L) TO (%L)',
                       partition_name, start_date, end_date);
    EXCEPTION WHEN duplicate_table THEN
        -- Partition already exists, continue
        NULL;
    END;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Note: To enable partitioning, you would need to:
-- 1. Create audit_logs as a partitioned table: CREATE TABLE audit_logs (...) PARTITION BY RANGE (created_at);
-- 2. Create the trigger: CREATE TRIGGER audit_partition_trigger BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION create_monthly_audit_partition();
-- This is commented out as it requires recreating the table, which should be done carefully in production