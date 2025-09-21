# 🧪 Secure File Vault - Complete Testing Guide

This guide provides comprehensive testing steps for all implemented features in the Secure File Vault system.

## 🚀 Quick Start Testing

### Step 1: Environment Setup
```bash
# Run the setup script
./scripts/setup-testing.sh

# Start all services
docker-compose up -d

# Wait for all services to be ready (especially ClamAV - takes 5-10 minutes)
docker-compose logs -f clamav
```

### Step 2: Basic Feature Testing
```bash
# Run basic feature tests
./scripts/test-features.sh
```

### Step 3: Advanced Security Testing
```bash
# Run advanced security and performance tests
./scripts/test-advanced.sh
```

## 📋 Manual Testing Checklist

### ✅ Authentication & Security
- [ ] User registration with email validation
- [ ] User login with JWT token generation
- [ ] CSRF token validation on state-changing requests
- [ ] Rate limiting on rapid requests (2 RPS, burst 5)
- [ ] Secure headers in HTTP responses
- [ ] Session management and token refresh

### ✅ File Operations
- [ ] File upload session creation
- [ ] Multipart file upload with virus scanning
- [ ] File download with SAS URL generation
- [ ] File deletion with permission checks
- [ ] Storage quota enforcement
- [ ] File metadata management

### ✅ Infrastructure Services
- [ ] PostgreSQL database connectivity
- [ ] Redis caching and rate limiting storage
- [ ] ClamAV virus scanning integration
- [ ] Azurite (Azure Storage Emulator)
- [ ] Prometheus metrics collection
- [ ] Grafana dashboard visualization

### ✅ Security Features
- [ ] Audit logging for all user actions
- [ ] Virus scanning on file uploads
- [ ] CSRF protection on forms
- [ ] SQL injection prevention
- [ ] XSS protection headers
- [ ] Clickjacking prevention

## 🔧 Service URLs for Testing

| Service | URL | Credentials |
|---------|-----|-------------|
| Backend API | http://localhost:8080 | - |
| Frontend | http://localhost:3000 | - |
| Grafana | http://localhost:3001 | admin/admin123 |
| Prometheus | http://localhost:9090 | - |
| Azurite Blob | http://localhost:10000 | devstoreaccount1 |

## 🧪 Detailed Testing Procedures

### Authentication Flow Testing
```bash
# 1. Get CSRF token
curl -X GET http://localhost:8080/csrf-token

# 2. Register new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: YOUR_CSRF_TOKEN" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123",
    "firstName": "Test",
    "lastName": "User"
  }'

# 3. Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: YOUR_CSRF_TOKEN" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123"
  }'
```

### File Upload Testing
```bash
# 1. Create upload session
curl -X POST http://localhost:8080/api/v1/files/upload-session \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "X-CSRF-Token: YOUR_CSRF_TOKEN" \
  -d '{
    "fileName": "test.txt",
    "fileSize": 1024
  }'

# 2. Upload file
curl -X POST http://localhost:8080/api/v1/files/upload/SESSION_TOKEN \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "X-CSRF-Token: YOUR_CSRF_TOKEN" \
  -F "file=@test.txt"
```

### Rate Limiting Testing
```bash
# Send rapid requests to trigger rate limiting
for i in {1..10}; do
  curl -X GET http://localhost:8080/api/v1/profile \
    -H "Authorization: Bearer YOUR_JWT_TOKEN" \
    -H "X-CSRF-Token: YOUR_CSRF_TOKEN"
  sleep 0.1
done
```

## 📊 Database Verification

### Check Audit Logs
```sql
-- Connect to PostgreSQL
docker exec -it soter-postgres psql -U postgres -d soter

-- View recent audit logs
SELECT action, resource_type, status, created_at 
FROM audit_logs 
ORDER BY created_at DESC 
LIMIT 10;

-- Check rate limiting data
SELECT * FROM user_rate_limits ORDER BY updated_at DESC LIMIT 5;

-- View file uploads
SELECT filename, file_size, upload_status, created_at 
FROM files 
ORDER BY created_at DESC 
LIMIT 10;
```

## 🐛 Troubleshooting

### Common Issues

1. **ClamAV Not Ready**
   ```bash
   # Check ClamAV logs
   docker-compose logs clamav
   
   # ClamAV takes 5-10 minutes to download virus definitions
   # Wait for: "Received 0 databases, need 3"
   ```

2. **Database Connection Issues**
   ```bash
   # Check PostgreSQL logs
   docker-compose logs postgres
   
   # Verify connection
   docker exec soter-postgres pg_isready -U postgres
   ```

3. **Redis Connection Issues**
   ```bash
   # Test Redis connectivity
   docker exec soter-redis redis-cli ping
   ```

4. **Rate Limiting Not Working**
   - Check Redis is running and accessible
   - Verify rate limiting configuration in .env
   - Check backend logs for rate limiting messages

### Service Health Checks
```bash
# Check all services
docker-compose ps

# View service logs
docker-compose logs [service-name]

# Health check endpoints
curl http://localhost:8080/health
curl http://localhost:8080/healthz
```

## 🎯 Performance Testing

### Load Testing with Apache Bench
```bash
# Install Apache Bench
sudo apt-get install apache2-utils  # Ubuntu/Debian
# or
brew install httpie  # macOS

# Test API endpoint
ab -n 1000 -c 10 -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/api/v1/profile
```

### Memory and CPU Monitoring
```bash
# Monitor Docker container resources
docker stats

# Check backend performance
curl http://localhost:8080/metrics | grep -E "memory|cpu|requests"
```

## 📈 Success Criteria

### ✅ All Tests Pass When:
- [ ] All Docker services start without errors
- [ ] Database migrations complete successfully
- [ ] Authentication flow works end-to-end
- [ ] File upload/download operations succeed
- [ ] Rate limiting triggers as expected
- [ ] Audit logs capture all actions
- [ ] Security headers are present
- [ ] Virus scanning detects test threats
- [ ] Performance metrics are collected
- [ ] CSRF protection blocks unauthorized requests

### 🚀 Production Readiness Indicators:
- [ ] < 2 second API response times
- [ ] Rate limiting prevents abuse
- [ ] Audit trail for compliance
- [ ] File virus scanning active
- [ ] Secure token management
- [ ] Proper error handling
- [ ] Monitoring and alerting configured

---

## 🎉 Congratulations!

If all tests pass, your Secure File Vault system is ready for production deployment with enterprise-grade security, monitoring, and performance features!