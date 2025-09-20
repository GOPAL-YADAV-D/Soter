-- Migration: Update schema for authentication system
-- This migration updates the existing schema to support proper authentication

-- Add name field to users table
ALTER TABLE users ADD COLUMN name VARCHAR(100) NOT NULL DEFAULT '';

-- Add created_by_user_id to organizations table
ALTER TABLE organizations ADD COLUMN created_by_user_id UUID REFERENCES users(id);

-- Create enum for user roles
CREATE TYPE user_role AS ENUM ('ADMIN', 'MEMBER', 'VIEWER');

-- Update user_organizations table to use enum
ALTER TABLE user_organizations ALTER COLUMN role TYPE user_role USING role::user_role;

-- Create refresh_tokens table for JWT management
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    is_revoked BOOLEAN DEFAULT FALSE
);

-- Create indexes for performance
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens(is_revoked);

-- Create index for organization creator lookup
CREATE INDEX idx_organizations_created_by ON organizations(created_by_user_id);

-- Update the default admin user to have a name
UPDATE users SET name = 'System Administrator' WHERE username = 'admin';

-- Create function to automatically set organization creator as admin
CREATE OR REPLACE FUNCTION set_organization_creator_as_admin()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert the creator as admin in user_organizations
    INSERT INTO user_organizations (user_id, organization_id, role)
    VALUES (NEW.created_by_user_id, NEW.id, 'ADMIN')
    ON CONFLICT (user_id, organization_id) DO NOTHING;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically set creator as admin
CREATE TRIGGER set_creator_as_admin_trigger
    AFTER INSERT ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION set_organization_creator_as_admin();

-- Create function to clean up expired refresh tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM refresh_tokens 
    WHERE expires_at < NOW() OR is_revoked = TRUE;
END;
$$ language 'plpgsql';

-- Add constraint to ensure organization names are unique and not empty
ALTER TABLE organizations ADD CONSTRAINT organizations_name_not_empty CHECK (LENGTH(TRIM(name)) > 0);

-- Add constraint to ensure user names are not empty
ALTER TABLE users ADD CONSTRAINT users_name_not_empty CHECK (LENGTH(TRIM(name)) > 0);

-- Add constraint to ensure email format is valid (basic check)
ALTER TABLE users ADD CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

