#!/bin/bash

# Quick verification script to check if the system is ready for testing

echo "🔍 Soter System Verification"
echo "============================"

# Check if all necessary files exist
FILES=(
    "docker-compose.yml"
    "backend/server"
    "scripts/setup-testing.sh"
    "scripts/test-features.sh"
    "TESTING_GUIDE.md"
)

echo "📁 Checking required files..."
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ $file (missing)"
    fi
done

echo ""

# Check if Docker is running
echo "🐳 Checking Docker..."
if docker info > /dev/null 2>&1; then
    echo "✅ Docker is running"
else
    echo "❌ Docker is not running - please start Docker"
    exit 1
fi

# Check if ports are available
echo ""
echo "🌐 Checking port availability..."
PORTS=(3000 8080 5433 6379 9090 3001 10000 3310)
for port in "${PORTS[@]}"; do
    if ss -tulpn | grep -q ":$port "; then
        echo "⚠️  Port $port is in use"
    else
        echo "✅ Port $port is available"
    fi
done

echo ""
echo "🎯 Next Steps:"
echo "1. Run: ./scripts/setup-testing.sh"
echo "2. Run: docker-compose up -d"
echo "3. Wait 5-10 minutes for ClamAV to initialize"
echo "4. Run: ./scripts/test-features.sh"
echo ""
echo "📚 See TESTING_GUIDE.md for detailed testing procedures"