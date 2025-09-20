# 🚀 Environment Configuration Guide for Soter

This guide explains how to configure your Secure File Vault System for different environments and how to switch between local development and production Azure services.

## 📁 Environment Files

- **`.env`** - Main environment configuration (for local development)
- **`.env.example`** - Template with safe default values
- **`.env.production`** - Production environment variables (create this for production)

## 🔧 Environment Variables Explained

### 1. Storage Configuration (`STORAGE_ENVIRONMENT`)

The `STORAGE_ENVIRONMENT` variable controls whether you use local Azurite or production Azure Blob Storage:

```bash
# For local development (uses Azurite)
STORAGE_ENVIRONMENT=local

# For production (uses Azure Blob Storage)
STORAGE_ENVIRONMENT=production
```

### 2. Local Development with Azurite

When `STORAGE_ENVIRONMENT=local`, the system uses these variables:

```bash
AZURITE_STORAGE_ACCOUNT=devstoreaccount1
AZURITE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
AZURITE_STORAGE_CONTAINER=files
AZURITE_STORAGE_ENDPOINT=http://localhost:10000/devstoreaccount1
```

**✅ You don't need to change these values for local development!**

### 3. Production with Azure Blob Storage

When `STORAGE_ENVIRONMENT=production`, update these variables with your Azure Storage account details:

```bash
AZURE_STORAGE_ACCOUNT=your-storage-account-name
AZURE_STORAGE_KEY=your-storage-account-key
AZURE_STORAGE_CONTAINER=files
AZURE_STORAGE_ENDPOINT=https://your-storage-account-name.blob.core.windows.net
```

**🔑 How to get your Azure Storage credentials:**

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to your Storage Account
3. Go to **Security + networking** > **Access keys**
4. Copy **Storage account name** and **Key1** or **Key2**

### 4. Database Configuration

#### Local Development (Docker PostgreSQL)
```bash
DB_HOST=localhost
DB_PORT=5433  # We use 5433 to avoid conflicts with local PostgreSQL
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=soter
DB_SSLMODE=disable
```

#### Production (Azure Database for PostgreSQL)
```bash
DB_HOST=your-postgres-server.postgres.database.azure.com
DB_PORT=5432
DB_USER=your-username@your-postgres-server
DB_PASSWORD=your-strong-password
DB_NAME=soter
DB_SSLMODE=require
```

## 🐳 Azurite Setup for Local Development

Azurite is already configured in your `docker-compose.yml`. Here's how to manage it:

### Start Azurite
```bash
# Start only Azurite
docker-compose up azurite -d

# Start Azurite + PostgreSQL
docker-compose up postgres azurite -d

# Start all services
docker-compose up -d
```

### Verify Azurite is Running
```bash
# Check if containers are running
docker ps

# Test Azurite endpoint
curl http://localhost:10000/devstoreaccount1

# Check Azurite logs
docker logs soter-azurite
```

### Azurite Web Interface
- **Blob Service**: http://localhost:10000
- **Queue Service**: http://localhost:10001  
- **Table Service**: http://localhost:10002

## 🔄 Switching Between Environments

### Method 1: Change Environment Variable
```bash
# Edit .env file
nano .env

# Change this line:
STORAGE_ENVIRONMENT=local    # For local development
STORAGE_ENVIRONMENT=production  # For production
```

### Method 2: Multiple Environment Files
```bash
# Create production environment file
cp .env .env.production

# Edit production file
nano .env.production
# Set STORAGE_ENVIRONMENT=production
# Update AZURE_STORAGE_* variables

# Use production environment
cp .env.production .env
```

### Method 3: Environment Override
```bash
# Override environment variable when running
STORAGE_ENVIRONMENT=production go run cmd/server/main.go
```

## 🎯 Quick Setup Guide

### For Local Development:
1. **Keep defaults in `.env`**:
   ```bash
   STORAGE_ENVIRONMENT=local
   DB_HOST=localhost
   DB_PORT=5433
   ```

2. **Start services**:
   ```bash
   docker-compose up postgres azurite -d
   ```

3. **Run application**:
   ```bash
   # Backend
   cd backend && go run cmd/server/main.go
   
   # Frontend (new terminal)
   cd frontend && npm start
   ```

### For Production:
1. **Update `.env` for production**:
   ```bash
   STORAGE_ENVIRONMENT=production
   AZURE_STORAGE_ACCOUNT=your-account-name
   AZURE_STORAGE_KEY=your-account-key
   DB_HOST=your-azure-postgres.postgres.database.azure.com
   DB_SSLMODE=require
   JWT_SECRET=your-super-secure-production-secret
   ```

2. **Deploy to production environment**

## 🔒 Security Best Practices

### JWT Secrets
```bash
# Development (can be simple)
JWT_SECRET=dev-jwt-secret-key

# Production (must be strong)
JWT_SECRET=your-super-secure-256-bit-random-key-generated-securely
```

### Generate Strong JWT Secret
```bash
# Using OpenSSL
openssl rand -base64 32

# Using Node.js
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

## 📊 Environment Verification

Use these commands to verify your environment:

```bash
# Check if backend can connect to database
curl http://localhost:8080/healthz

# Check if Azurite is accessible
curl http://localhost:10000/devstoreaccount1

# Test authentication endpoint
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'
```

## 🚨 Common Issues & Solutions

### Issue: "Connection refused" to database
**Solution**: Make sure PostgreSQL container is running
```bash
docker-compose up postgres -d
```

### Issue: "Connection refused" to Azurite
**Solution**: Make sure Azurite container is running
```bash
docker-compose up azurite -d
```

### Issue: Port 5432 already in use
**Solution**: We're using port 5433 for Docker PostgreSQL to avoid conflicts

### Issue: Backend can't find environment variables
**Solution**: Make sure you're running from the project root or backend directory

## 🔄 Environment Variable Precedence

1. **System environment variables** (highest priority)
2. **`.env` file in current directory**
3. **Default values in code** (lowest priority)

## 📝 Your Azure Setup Checklist

When you're ready for production:

- [ ] Create Azure Storage Account
- [ ] Get storage account name and access key
- [ ] Create blob container named "files"
- [ ] Update AZURE_STORAGE_* variables in .env
- [ ] Set STORAGE_ENVIRONMENT=production
- [ ] Create Azure Database for PostgreSQL
- [ ] Update DB_* variables for production database
- [ ] Generate strong JWT secret for production
- [ ] Test connection to Azure services

## 🎉 Current Status

Your system is currently configured for:
- ✅ **Local Development** with Azurite
- ✅ **Local PostgreSQL** (Docker, port 5433)
- ✅ **Authentication** with JWT
- ✅ **CORS** enabled for frontend
- ✅ **Health checks** available

**Ready to use!** 🚀

Just update the environment variables when you're ready to switch to production Azure services.