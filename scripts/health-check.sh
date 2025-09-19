#!/bin/bash

# Soter Health Check Script
# Performs comprehensive health checks for all services

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Soter Health Check"
echo "===================="

# Function to check HTTP endpoint
check_endpoint() {
    local url=$1
    local name=$2
    
    if curl -f -s "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ $name${NC}"
        return 0
    else
        echo -e "${RED}❌ $name${NC}"
        return 1
    fi
}

# Function to check Docker container
check_container() {
    local container=$1
    local name=$2
    
    if docker-compose ps -q "$container" | xargs docker inspect --format='{{.State.Health.Status}}' 2>/dev/null | grep -q "healthy"; then
        echo -e "${GREEN}✅ $name Container${NC}"
        return 0
    elif docker-compose ps -q "$container" > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  $name Container (running but not healthy)${NC}"
        return 1
    else
        echo -e "${RED}❌ $name Container${NC}"
        return 1
    fi
}

overall_status=0

echo ""
echo "🐳 Container Health:"
check_container "postgres" "PostgreSQL" || overall_status=1
check_container "azurite" "Azurite" || overall_status=1
check_container "backend" "Backend" || overall_status=1
check_container "frontend" "Frontend" || overall_status=1
check_container "prometheus" "Prometheus" || overall_status=1
check_container "grafana" "Grafana" || overall_status=1

echo ""
echo "🌐 Endpoint Health:"
check_endpoint "http://localhost:8080/healthz" "Backend API" || overall_status=1
check_endpoint "http://localhost:8080/metrics" "Metrics Endpoint" || overall_status=1
check_endpoint "http://localhost:3000" "Frontend" || overall_status=1
check_endpoint "http://localhost:9090" "Prometheus" || overall_status=1
check_endpoint "http://localhost:3001" "Grafana" || overall_status=1

echo ""
echo "📊 Database Connectivity:"
if docker-compose exec -T postgres pg_isready -U postgres -d soter > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PostgreSQL Connection${NC}"
else
    echo -e "${RED}❌ PostgreSQL Connection${NC}"
    overall_status=1
fi

echo ""
echo "💾 Storage Connectivity:"
if curl -f -s "http://localhost:10000/devstoreaccount1" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Azurite Storage${NC}"
else
    echo -e "${RED}❌ Azurite Storage${NC}"
    overall_status=1
fi

echo ""
echo "===================="
if [ $overall_status -eq 0 ]; then
    echo -e "${GREEN}🎉 All systems healthy!${NC}"
else
    echo -e "${RED}⚠️  Some issues detected. Check logs:${NC}"
    echo "   docker-compose logs [service-name]"
fi

exit $overall_status