#!/bin/bash

set -e

echo "🚀 Setting up WhatsApp Recovery System Local Environment"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check Docker
echo "📦 Checking Docker..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker is installed${NC}"

# Check Docker Compose
echo "📦 Checking Docker Compose..."
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed. Please install Docker Compose first.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker Compose is installed${NC}"

# Determine docker compose command
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo ""
echo "🏗️  Building Docker images..."
cd "$(dirname "$0")/../../docker" || exit 1

$DOCKER_COMPOSE build

echo ""
echo -e "${GREEN}✅ Build completed successfully${NC}"
echo ""

echo "🚀 Starting services..."
$DOCKER_COMPOSE up -d

echo ""
echo "⏳ Waiting for services to be healthy..."
sleep 5

# Wait for health checks (max 30 seconds)
TIMEOUT=30
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if $DOCKER_COMPOSE ps | grep -q "unhealthy"; then
        echo -e "${YELLOW}⏳ Waiting for services to become healthy...${NC}"
        sleep 5
        ELAPSED=$((ELAPSED + 5))
    else
        break
    fi
done

echo ""
echo "📊 Service Status:"
$DOCKER_COMPOSE ps

echo ""
echo -e "${GREEN}✅ Setup complete!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📱 WhatsApp Recovery System - Local Environment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🔗 Service Endpoints:"
echo "  Event Tracker API:  http://localhost:8080"
echo "  AppConfig Mock:     http://localhost:2772"
echo "  WhatsApp Mock:      http://localhost:8081"
echo "  Redis:              localhost:6379"
echo ""
echo "📝 Quick Commands:"
echo "  View logs:          cd docker && $DOCKER_COMPOSE logs -f"
echo "  View worker logs:   cd docker && $DOCKER_COMPOSE logs -f worker"
echo "  Restart worker:     cd docker && $DOCKER_COMPOSE restart worker"
echo "  Stop all:           cd docker && $DOCKER_COMPOSE down"
echo "  Clean slate:        cd docker && $DOCKER_COMPOSE down -v"
echo ""
echo "🧪 Test API:"
echo "  curl -X POST http://localhost:8080/journey/event \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{"
echo "      \"journey_id\": \"onboarding-v2\","
echo "      \"step\": \"personal-data\","
echo "      \"customer_number\": \"5511999999999\","
echo "      \"tenant_id\": \"tenant-123\","
echo "      \"contact_id\": \"contact-456\","
echo "      \"metadata\": {"
echo "        \"nome_cliente\": \"João Silva\""
echo "      }"
echo "    }'"
echo ""
echo "🔍 Inspect Redis:"
echo "  cd docker && $DOCKER_COMPOSE exec redis redis-cli"
echo "  > KEYS journey:*"
echo "  > GET journey:onboarding-v2:5511999999999:state"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📚 For more information, see local/README.md"
echo ""
