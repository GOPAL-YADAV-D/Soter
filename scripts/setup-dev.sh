#!/bin/bash

# Soter Development Setup Script
# This script helps set up the development environment for Soter

set -e

echo "🚀 Setting up Soter development environment..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Check if Go is installed (for backend development)
if ! command -v go &> /dev/null; then
    echo "⚠️  Go is not installed. Backend development will require Go 1.21+"
fi

# Check if Node.js is installed (for frontend development)
if ! command -v node &> /dev/null; then
    echo "⚠️  Node.js is not installed. Frontend development will require Node.js 18+"
fi

echo "📦 Starting services with Docker Compose..."
docker-compose up -d

echo "⏳ Waiting for services to be ready..."
sleep 30

# Check if backend is running
echo "🔍 Checking backend health..."
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
    echo "✅ Backend is running and healthy"
else
    echo "❌ Backend health check failed"
    docker-compose logs backend
    exit 1
fi

# Check if frontend is running
echo "🔍 Checking frontend..."
if curl -f http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend is running"
else
    echo "❌ Frontend check failed"
    docker-compose logs frontend
    exit 1
fi

echo ""
echo "🎉 Soter is now running!"
echo ""
echo "🌐 Access points:"
echo "   Frontend:          http://localhost:3000"
echo "   GraphQL Playground: http://localhost:8080/playground"
echo "   Health Check:      http://localhost:8080/healthz"
echo "   Metrics:           http://localhost:8080/metrics"
echo "   Prometheus:        http://localhost:9090"
echo "   Grafana:           http://localhost:3001 (admin/admin123)"
echo ""
echo "📋 Default credentials:"
echo "   Admin: admin@soter.local / admin123"
echo ""
echo "🛑 To stop: docker-compose down"
echo "🗑️  To reset: docker-compose down -v"