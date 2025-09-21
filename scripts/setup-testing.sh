#!/bin/bash

# Secure File Vault - Testing Setup Script
# This script sets up the development environment for testing

set -e

echo "🚀 Setting up Secure File Vault for testing..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose > /dev/null 2>&1 && ! docker compose version > /dev/null 2>&1; then
    print_error "Docker Compose is not available. Please install Docker Compose."
    exit 1
fi

# Create necessary directories
print_status "Creating storage directories..."
mkdir -p backend/storage/files
mkdir -p backend/logs
mkdir -p infra/postgres/data

# Set proper permissions
chmod 755 backend/storage
chmod 755 backend/logs

# Create .env file if it doesn't exist
if [ ! -f ".env" ]; then
    print_status "Creating .env file..."
    cat > .env << EOF
# Database Configuration
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=soter
DB_SSLMODE=disable

# Storage Configuration
STORAGE_ENVIRONMENT=local
LOCAL_STORAGE_PATH=./storage
STORAGE_QUOTA_MB=100

# Azure Storage (for production)
AZURE_STORAGE_ACCOUNT=devstoreaccount1
AZURE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
AZURE_STORAGE_CONTAINER=files
AZURE_STORAGE_ENDPOINT=http://localhost:10000/devstoreaccount1

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-12345

# Rate Limiting
RATE_LIMIT_RPS=2
RATE_LIMIT_BURST=5

# Security
CSRF_SECRET=csrf-secret-change-in-production-67890
ENABLE_VIRUS_SCANNING=true

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis-password
REDIS_DB=0

# ClamAV Configuration
CLAMAV_HOST=localhost
CLAMAV_PORT=3310

# Application
PORT=8080
HOST=0.0.0.0
LOG_LEVEL=info
GIN_MODE=debug
EOF
    print_status ".env file created successfully!"
else
    print_warning ".env file already exists, skipping..."
fi

# Build Go backend
print_status "Building Go backend..."
cd backend
if go build -o server ./cmd/server; then
    print_status "Backend built successfully!"
else
    print_error "Backend build failed!"
    exit 1
fi
cd ..

# Build frontend
print_status "Building React frontend..."
cd frontend
if [ -f "package.json" ]; then
    if command -v npm > /dev/null 2>&1; then
        npm install
        npm run build
        print_status "Frontend built successfully!"
    else
        print_warning "npm not found, skipping frontend build"
    fi
else
    print_warning "package.json not found, skipping frontend build"
fi
cd ..

print_status "Setup complete! 🎉"
echo ""
echo "Next steps:"
echo "1. Start services: docker-compose up -d"
echo "2. Wait for ClamAV to download virus definitions (5-10 minutes)"
echo "3. Run the test script: ./scripts/test-features.sh"
echo ""
echo "Service URLs:"
echo "- Backend API: http://localhost:8080"
echo "- Frontend: http://localhost:3000"
echo "- Grafana: http://localhost:3001 (admin/admin123)"
echo "- Prometheus: http://localhost:9090"
echo "- Azurite: http://localhost:10000"