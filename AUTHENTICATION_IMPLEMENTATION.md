# Authentication System Implementation

## Overview

This document describes the comprehensive authentication system implemented for the Secure File Vault System (Soter). The system provides JWT-based authentication with role-based access control (RBAC) and organization management.

## Architecture

### Core Components

1. **Authentication Service** (`internal/services/auth.go`)
   - Password hashing using Argon2id
   - JWT token generation and validation
   - Refresh token management
   - User authentication logic

2. **User Repository** (`internal/repositories/user.go`)
   - Database operations for users and organizations
   - Transaction management for user-organization creation
   - Data validation and existence checks

3. **Authentication Middleware** (`internal/middleware/auth.go`)
   - JWT token validation
   - Role-based access control
   - Context management for authenticated users

4. **GraphQL Resolvers** (`graph/resolvers.go`)
   - Authentication mutations and queries
   - Input validation
   - Error handling

5. **Models** (`internal/models/user.go`)
   - User, Organization, and authentication data structures
   - Role definitions and permission logic

## Database Schema

### Updated Tables

#### Users Table
```sql
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
```

#### Organizations Table
```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_by_user_id UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);
```

#### User Organizations Junction Table
```sql
CREATE TABLE user_organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role user_role DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, organization_id)
);
```

#### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    is_revoked BOOLEAN DEFAULT FALSE
);
```

## Authentication Flow

### 1. User Registration with Organization

**GraphQL Mutation:**
```graphql
mutation RegisterUserWithOrganization(
  $name: String!
  $username: String!
  $email: String!
  $password: String!
  $organizationName: String!
  $organizationDescription: String
) {
  registerUserWithOrganization(
    name: $name
    username: $username
    email: $email
    password: $password
    organizationName: $organizationName
    organizationDescription: $organizationDescription
  ) {
    tokens {
      accessToken
      refreshToken
      expiresIn
    }
    user {
      id
      name
      email
      role
      organization {
        id
        name
      }
    }
  }
}
```

**Process:**
1. Validate input data (email format, password strength, etc.)
2. Check for existing email/username/organization name
3. Hash password using Argon2id
4. Create user and organization in a database transaction
5. Automatically assign ADMIN role to the creator
6. Generate JWT access token and refresh token
7. Return authentication response

### 2. User Login

**GraphQL Mutation:**
```graphql
mutation LoginUser($email: String!, $password: String!) {
  loginUser(email: $email, password: $password) {
    tokens {
      accessToken
      refreshToken
      expiresIn
    }
    user {
      id
      name
      email
      role
      organization {
        id
        name
      }
    }
  }
}
```

**Process:**
1. Validate email and password
2. Find user by email
3. Verify password using Argon2id
4. Update last login timestamp
5. Generate new token pair
6. Return authentication response

### 3. Token Refresh

**GraphQL Mutation:**
```graphql
mutation RefreshToken($refreshToken: String!) {
  refreshToken(refreshToken: $refreshToken) {
    accessToken
    refreshToken
    expiresIn
  }
}
```

**Process:**
1. Validate refresh token
2. Check token expiration and revocation status
3. Generate new access token
4. Optionally generate new refresh token
5. Return new token pair

### 4. User Logout

**GraphQL Mutation:**
```graphql
mutation LogoutUser($refreshToken: String!) {
  logoutUser(refreshToken: $refreshToken)
}
```

**Process:**
1. Revoke the provided refresh token
2. Mark token as revoked in database
3. Return success status

## Role-Based Access Control (RBAC)

### Role Hierarchy

1. **ADMIN** - Full access to organization
   - Can manage organization settings
   - Can invite/remove users
   - Can upload, download, and manage files
   - Can view all organization data

2. **MEMBER** - Standard user access
   - Can upload and download files
   - Can view organization files
   - Cannot manage organization settings

3. **VIEWER** - Read-only access
   - Can view files
   - Cannot upload or modify files
   - Cannot manage organization settings

### Permission Methods

```go
// Check if role has required permission level
func (r UserRole) HasPermission(required UserRole) bool

// Check if role can manage organization
func (r UserRole) CanManageOrganization() bool

// Check if role can upload files
func (r UserRole) CanUploadFiles() bool

// Check if role can view files
func (r UserRole) CanViewFiles() bool
```

## Security Features

### Password Security
- **Argon2id** hashing algorithm
- Configurable memory, iterations, and parallelism
- Salt generation for each password
- Protection against timing attacks

### JWT Security
- **HMAC-SHA256** signing algorithm
- Short-lived access tokens (1 hour)
- Long-lived refresh tokens (7 days)
- Token revocation support
- Secure token storage in database

### Input Validation
- Email format validation
- Password strength requirements
- Username format validation
- Organization name validation
- SQL injection protection through parameterized queries

### Middleware Security
- Authentication middleware for protected routes
- Role-based authorization middleware
- Optional authentication for public routes
- Request ID tracking for audit logs

## API Endpoints

### GraphQL Endpoints
- `POST /query` - GraphQL endpoint with authentication middleware
- `GET /playground` - GraphQL playground for development

### REST Endpoints
- `GET /healthz` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint

## Usage Examples

### 1. Register a New User with Organization

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation RegisterUserWithOrganization($name: String!, $username: String!, $email: String!, $password: String!, $organizationName: String!) { registerUserWithOrganization(name: $name, username: $username, email: $email, password: $password, organizationName: $organizationName) { tokens { accessToken refreshToken expiresIn } user { id name email role organization { id name } } } }",
    "variables": {
      "name": "John Doe",
      "username": "johndoe",
      "email": "john@example.com",
      "password": "securepassword123",
      "organizationName": "Acme Corp"
    }
  }'
```

### 2. Login User

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation LoginUser($email: String!, $password: String!) { loginUser(email: $email, password: $password) { tokens { accessToken refreshToken expiresIn } user { id name email role organization { id name } } } }",
    "variables": {
      "email": "john@example.com",
      "password": "securepassword123"
    }
  }'
```

### 3. Get Current User (Requires Authentication)

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "query": "query Me { me { id name email role organization { id name } } }"
  }'
```

## Configuration

### Environment Variables

```bash
# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=soter
DB_SSLMODE=disable

# Server Configuration
PORT=8080
HOST=0.0.0.0
LOG_LEVEL=info
```

## Future Enhancements

### Planned Features
1. **Multi-factor Authentication (MFA)**
   - TOTP support
   - SMS verification
   - Email verification

2. **Advanced RBAC**
   - Custom roles
   - Resource-specific permissions
   - Time-based access controls

3. **Session Management**
   - Active session tracking
   - Remote logout capability
   - Session timeout management

4. **Audit Logging**
   - Comprehensive audit trail
   - Security event logging
   - Compliance reporting

### Security Improvements
1. **Rate Limiting**
   - Per-user rate limits
   - IP-based rate limiting
   - Adaptive rate limiting

2. **Security Headers**
   - CSRF protection
   - XSS protection
   - Content Security Policy

3. **Token Security**
   - Token rotation
   - Device-specific tokens
   - Token binding

## Testing

### Unit Tests
- Authentication service tests
- Password hashing tests
- JWT token tests
- Repository tests

### Integration Tests
- End-to-end authentication flow
- Database transaction tests
- Middleware tests

### Security Tests
- Password strength validation
- Token expiration tests
- Authorization tests
- Input validation tests

## Deployment Considerations

### Production Security
1. **Change Default Secrets**
   - Update JWT secret
   - Change database passwords
   - Update admin credentials

2. **Enable HTTPS**
   - SSL/TLS certificates
   - HTTP to HTTPS redirect
   - Secure cookie settings

3. **Database Security**
   - Enable SSL connections
   - Use connection pooling
   - Regular security updates

4. **Monitoring**
   - Authentication metrics
   - Failed login attempts
   - Token usage patterns
   - Security alerts

## Troubleshooting

### Common Issues

1. **Token Validation Errors**
   - Check JWT secret configuration
   - Verify token expiration
   - Check token format

2. **Database Connection Issues**
   - Verify database credentials
   - Check network connectivity
   - Validate SSL configuration

3. **Authentication Failures**
   - Check password hashing
   - Verify user status
   - Check organization membership

### Debug Mode
Enable debug logging by setting `LOG_LEVEL=debug` in environment variables.

## Conclusion

The authentication system provides a robust, secure foundation for the Secure File Vault System. It implements industry-standard security practices including Argon2id password hashing, JWT tokens, and role-based access control. The system is designed to be scalable, maintainable, and secure for production use.

The implementation follows Go best practices and provides comprehensive error handling, logging, and monitoring capabilities. Future enhancements can be easily integrated into the existing architecture.
