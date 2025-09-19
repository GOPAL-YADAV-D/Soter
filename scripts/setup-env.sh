#!/bin/bash

# Soter Environment Setup Script
# This script helps configure the environment for different deployment scenarios

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Soter Environment Configuration Setup${NC}"
echo "=========================================="

# Function to generate a secure JWT secret
generate_jwt_secret() {
    if command -v openssl &> /dev/null; then
        openssl rand -base64 64 | tr -d '\n'
    else
        # Fallback if openssl is not available
        date +%s | sha256sum | base64 | head -c 64
    fi
}

# Function to prompt for user input with default value
prompt_with_default() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"
    
    echo -n -e "${YELLOW}$prompt${NC} [${BLUE}$default${NC}]: "
    read -r user_input
    
    if [ -z "$user_input" ]; then
        user_input="$default"
    fi
    
    eval "$var_name='$user_input'"
}

# Check if .env already exists
if [ -f ".env" ]; then
    echo -e "${YELLOW}⚠️  .env file already exists!${NC}"
    echo -n "Do you want to overwrite it? (y/N): "
    read -r overwrite
    if [[ ! $overwrite =~ ^[Yy]$ ]]; then
        echo "Exiting without changes."
        exit 0
    fi
    echo "Backing up existing .env to .env.backup"
    cp .env .env.backup
fi

echo ""
echo "Please choose your deployment environment:"
echo "1) Local Development (with Azurite)"
echo "2) Staging/Testing"
echo "3) Production (with Azure Blob Storage)"
echo ""
echo -n "Enter your choice (1-3): "
read -r env_choice

case $env_choice in
    1)
        ENV_TYPE="development"
        echo -e "${GREEN}Setting up for Local Development${NC}"
        ;;
    2)
        ENV_TYPE="staging"
        echo -e "${YELLOW}Setting up for Staging/Testing${NC}"
        ;;
    3)
        ENV_TYPE="production"
        echo -e "${RED}Setting up for Production${NC}"
        ;;
    *)
        echo "Invalid choice. Defaulting to Local Development."
        ENV_TYPE="development"
        ;;
esac

echo ""
echo "Configuring environment variables..."

# Generate secure JWT secret
JWT_SECRET=$(generate_jwt_secret)

# Basic server configuration
prompt_with_default "Server Host" "0.0.0.0" "HOST"
prompt_with_default "Server Port" "8080" "PORT"
prompt_with_default "Log Level (debug/info/warn/error)" "info" "LOG_LEVEL"

echo ""
echo "Database Configuration:"
prompt_with_default "PostgreSQL Host" "localhost" "DB_HOST"
prompt_with_default "PostgreSQL Port" "5432" "DB_PORT"
prompt_with_default "PostgreSQL Username" "postgres" "DB_USER"
prompt_with_default "PostgreSQL Password" "password" "DB_PASSWORD"
prompt_with_default "PostgreSQL Database Name" "soter" "DB_NAME"

if [ "$ENV_TYPE" = "production" ]; then
    prompt_with_default "PostgreSQL SSL Mode" "require" "DB_SSLMODE"
else
    DB_SSLMODE="disable"
fi

echo ""
if [ "$ENV_TYPE" = "development" ]; then
    echo "Using Azurite (Local Azure Storage Emulator)"
    AZURE_STORAGE_ACCOUNT="devstoreaccount1"
    AZURE_STORAGE_KEY="Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
    AZURE_STORAGE_ENDPOINT="http://localhost:10000/devstoreaccount1"
    AZURE_STORAGE_CONTAINER="files"
else
    echo "Azure Blob Storage Configuration:"
    prompt_with_default "Azure Storage Account Name" "your-storage-account" "AZURE_STORAGE_ACCOUNT"
    prompt_with_default "Azure Storage Container" "files" "AZURE_STORAGE_CONTAINER"
    
    echo ""
    echo -e "${YELLOW}Important: You need to manually set your Azure Storage Key${NC}"
    echo "Get it from: Azure Portal > Storage Account > Access Keys"
    AZURE_STORAGE_KEY="your-azure-storage-key-here"
    AZURE_STORAGE_ENDPOINT=""
fi

echo ""
echo "Security Configuration:"
prompt_with_default "Storage Quota per User (MB)" "10" "STORAGE_QUOTA_MB"
prompt_with_default "Rate Limit (requests per second)" "2" "RATE_LIMIT_RPS"
prompt_with_default "Rate Limit Burst" "5" "RATE_LIMIT_BURST"

# Create the .env file
cat > .env << EOF
# Soter - ${ENV_TYPE^} Environment Configuration
# Generated on $(date)
# =============================================================

# =============================================================
# SERVER CONFIGURATION
# =============================================================
HOST=$HOST
PORT=$PORT
NODE_ENV=$ENV_TYPE
GO_ENV=$ENV_TYPE
LOG_LEVEL=$LOG_LEVEL

# =============================================================
# DATABASE CONFIGURATION (PostgreSQL)
# =============================================================
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME
DB_SSLMODE=$DB_SSLMODE
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=5m
TEST_DB_NAME=soter_test

# =============================================================
# AZURE STORAGE CONFIGURATION
# =============================================================
AZURE_STORAGE_ACCOUNT=$AZURE_STORAGE_ACCOUNT
AZURE_STORAGE_KEY=$AZURE_STORAGE_KEY
AZURE_STORAGE_CONTAINER=$AZURE_STORAGE_CONTAINER
AZURE_STORAGE_ENDPOINT=$AZURE_STORAGE_ENDPOINT

# =============================================================
# SECURITY CONFIGURATION
# =============================================================
JWT_SECRET=$JWT_SECRET
JWT_EXPIRATION=24h
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001

# =============================================================
# RATE LIMITING & QUOTAS
# =============================================================
RATE_LIMIT_RPS=$RATE_LIMIT_RPS
RATE_LIMIT_BURST=$RATE_LIMIT_BURST
STORAGE_QUOTA_MB=$STORAGE_QUOTA_MB
MAX_FILE_SIZE_BYTES=104857600
MAX_FILES_PER_BATCH=10

# =============================================================
# MONITORING & OBSERVABILITY
# =============================================================
METRICS_ENABLED=true
HEALTH_CHECK_TIMEOUT=5s
REQUEST_TIMEOUT=30s
SHUTDOWN_TIMEOUT=30s

# =============================================================
# FRONTEND CONFIGURATION
# =============================================================
REACT_APP_API_URL=http://localhost:$PORT
REACT_APP_GRAPHQL_URL=http://localhost:$PORT/query
REACT_APP_VERSION=1.0.0

# =============================================================
# EXTERNAL SERVICES
# =============================================================
PROMETHEUS_URL=http://localhost:9090
GRAFANA_URL=http://localhost:3001
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin123

# =============================================================
# DEVELOPMENT & DEBUGGING
# =============================================================
DEBUG=false
VERBOSE_LOGGING=false
HOT_RELOAD=true
AUTO_MIGRATE=true
EOF

echo ""
echo -e "${GREEN}✅ Environment configuration created successfully!${NC}"
echo ""
echo "📁 Files created:"
echo "   📄 .env (main environment file)"
if [ -f ".env.backup" ]; then
    echo "   📄 .env.backup (backup of previous .env)"
fi

echo ""
echo -e "${YELLOW}🔐 Security Notes:${NC}"
echo "   • A secure JWT secret has been generated automatically"
if [ "$ENV_TYPE" != "development" ]; then
    echo "   • Remember to update AZURE_STORAGE_KEY with your actual Azure key"
    echo "   • Update DB_PASSWORD with a secure password"
fi
echo "   • Never commit .env files to version control"
echo "   • Review and update configuration as needed"

echo ""
echo -e "${BLUE}🚀 Next Steps:${NC}"
if [ "$ENV_TYPE" = "development" ]; then
    echo "   1. Start services: docker-compose up -d"
    echo "   2. Check health: ./scripts/health-check.sh"
    echo "   3. Access frontend: http://localhost:3000"
else
    echo "   1. Update Azure Storage credentials in .env"
    echo "   2. Update database password if needed"
    echo "   3. Review all configuration values"
    echo "   4. Deploy your application"
fi

echo ""
echo -e "${GREEN}Happy coding! 🎉${NC}"