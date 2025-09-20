# Environment Variables Guide

This guide explains all the environment variables used in the Secure File Vault System and where to configure them for different environments.

## 📍 Environment Files Location

The environment variables are configured in the following files:
- **Backend**: `/home/gopal/DataVault/Vault/Soter/.env`
- **Example Template**: `/home/gopal/DataVault/Vault/Soter/.env.example`

## 🔧 Backend Environment Variables

### Database Configuration
```bash
# PostgreSQL Database Settings
DB_HOST=localhost                    # Database host (use 'postgres' for Docker)
DB_PORT=5432                        # Database port
DB_USER=soter_user                  # Database username
DB_PASSWORD=secure_password_123     # Database password
DB_NAME=soter_vault                 # Database name
DB_SSLMODE=disable                  # SSL mode for database connection
```

### JWT Authentication
```bash
# JWT Configuration for Authentication
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_REFRESH_SECRET=your-super-secret-refresh-key-change-this-in-production
JWT_EXPIRES_IN=24h                  # Access token expiration (24 hours)
JWT_REFRESH_EXPIRES_IN=168h         # Refresh token expiration (7 days)
```

### Server Configuration
```bash
# Server Settings
PORT=8080                           # Backend server port
GIN_MODE=debug                      # Gin mode: debug, release, or test
LOG_LEVEL=info                      # Logging level: debug, info, warn, error
```

### Azure Blob Storage (Development uses Azurite)
```bash
# Azure Storage Configuration
AZURE_STORAGE_ACCOUNT=devstoreaccount1
AZURE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
AZURE_STORAGE_CONTAINER=files
AZURE_STORAGE_ENDPOINT=http://localhost:10000/devstoreaccount1
```

### Monitoring Configuration
```bash
# Prometheus Metrics
METRICS_ENABLED=true               # Enable/disable metrics collection
METRICS_PATH=/metrics              # Metrics endpoint path

# Health Check
HEALTH_CHECK_ENABLED=true          # Enable/disable health checks
HEALTH_CHECK_PATH=/healthz         # Health check endpoint path
```

## 🚀 How to Start the Project

### Prerequisites
1. **Docker & Docker Compose** installed
2. **Go 1.21+** installed
3. **Node.js 18+** and **npm** installed

### Step 1: Clone and Setup
```bash
# Navigate to project directory
cd /home/gopal/DataVault/Vault/Soter

# Copy environment template
cp .env.example .env

# Make setup script executable
chmod +x scripts/setup-dev.sh
```

### Step 2: Configure Environment Variables
Edit the `.env` file with your preferred settings:

```bash
# Open .env file for editing
nano .env

# Or use your preferred editor
code .env
```

**Important Variables to Change for Production:**
1. `JWT_SECRET` - Change to a strong, random secret
2. `JWT_REFRESH_SECRET` - Change to a different strong, random secret  
3. `DB_PASSWORD` - Use a strong database password
4. `GIN_MODE=release` - Set to release for production

### Step 3: Start the Development Environment
```bash
# Option 1: Use the setup script
./scripts/setup-dev.sh

# Option 2: Manual startup
docker-compose up -d

# Option 3: Start specific services
docker-compose up postgres azurite prometheus grafana -d
```

### Step 4: Build and Run Backend
```bash
# Navigate to backend directory
cd backend

# Install Go dependencies
go mod tidy

# Generate GraphQL code (if needed)
go run github.com/99designs/gqlgen generate

# Run the backend server
go run cmd/server/main.go
```

### Step 5: Build and Run Frontend
```bash
# Navigate to frontend directory (in a new terminal)
cd frontend

# Install dependencies
npm install

# Start development server
npm start
```

## 🌐 Application URLs

Once everything is running, you can access:

- **Frontend Application**: http://localhost:3000
- **Sign In Page**: http://localhost:3000/signin
- **Dashboard**: http://localhost:3000/dashboard
- **GraphQL Playground**: http://localhost:8080/playground
- **Backend Health Check**: http://localhost:8080/healthz
- **Prometheus Metrics**: http://localhost:8080/metrics
- **Grafana Dashboard**: http://localhost:3001 (admin/admin)

## 🔐 Authentication System

### Default Registration
When you first access the application:

1. **Go to**: http://localhost:3000/signin
2. **Click**: "Sign up" to create an account
3. **Fill in**:
   - Full Name: Your name
   - Username: Unique username
   - Email: Your email address
   - Password: Strong password
   - Organization Name: Your company/organization
   - Organization Description: Optional description

### Sign In
After registration, use your email and password to sign in.

## 🐳 Docker Services

The application runs the following Docker services:

| Service | Port | Description |
|---------|------|-------------|
| postgres | 5432 | PostgreSQL Database |
| azurite | 10000-10002 | Azure Blob Storage Emulator |
| prometheus | 9090 | Metrics Collection |
| grafana | 3001 | Metrics Visualization |

## 📊 Database Schema

The system automatically creates the following tables:
- `users` - User accounts
- `organizations` - Organization data
- `files` - File metadata
- `file_chunks` - File chunk storage
- `audit_logs` - Security audit trail
- `deduplication_hashes` - File deduplication

## 🔧 Common Environment Variable Changes

### For Local Development
```bash
# Database (when using Docker)
DB_HOST=localhost  # or 'postgres' if backend also runs in Docker

# JWT (development)
JWT_SECRET=dev-secret-key
JWT_REFRESH_SECRET=dev-refresh-secret-key

# Logging
LOG_LEVEL=debug
GIN_MODE=debug
```

### For Production
```bash
# Database
DB_HOST=your-production-db-host
DB_PASSWORD=your-strong-production-password
DB_SSLMODE=require

# JWT (production - use strong random keys)
JWT_SECRET=your-very-strong-random-jwt-secret-256-bits
JWT_REFRESH_SECRET=your-very-strong-random-refresh-secret-256-bits

# Server
GIN_MODE=release
LOG_LEVEL=warn

# Azure (production storage)
AZURE_STORAGE_ACCOUNT=your-storage-account
AZURE_STORAGE_KEY=your-storage-key
AZURE_STORAGE_ENDPOINT=https://your-account.blob.core.windows.net
```

## 🛠️ Troubleshooting

### Docker Permission Issues
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Restart Docker service
sudo systemctl restart docker

# Log out and log back in, or run:
newgrp docker
```

### Database Connection Issues
1. Check if PostgreSQL container is running: `docker ps`
2. Verify database credentials in `.env`
3. Check database logs: `docker logs soter_postgres_1`

### Frontend Build Issues
```bash
# Clear npm cache
npm cache clean --force

# Delete node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

### Backend Build Issues
```bash
# Clean Go module cache
go clean -modcache

# Reinstall dependencies
go mod tidy
go mod download
```

## 📝 Notes

- The `.env` file contains sensitive information and should never be committed to version control
- Always use strong, unique secrets for JWT tokens in production
- The Azurite storage emulator is for development only - use real Azure Blob Storage for production
- Monitor the application logs for any security events or errors

For additional help, check the application logs or refer to the project documentation.