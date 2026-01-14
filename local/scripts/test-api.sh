#!/bin/bash

# Helper script for testing the Event Tracker API

API_URL="http://localhost:8080"
REDIS_CONTAINER="recovery-redis"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Function to post an event
post_event() {
    local journey_id="${1:-onboarding-v2}"
    local step="${2:-personal-data}"
    local customer_number="${3:-5511999999999}"
    local nome_cliente="${4:-João Silva}"

    echo -e "${BLUE}📤 Posting event...${NC}"
    echo "  Journey: $journey_id"
    echo "  Step: $step"
    echo "  Customer: $customer_number"
    echo ""

    curl -X POST "$API_URL/journey/event" \
        -H "Content-Type: application/json" \
        -d "{
            \"journey_id\": \"$journey_id\",
            \"step\": \"$step\",
            \"customer_number\": \"$customer_number\",
            \"tenant_id\": \"tenant-123\",
            \"contact_id\": \"contact-456\",
            \"metadata\": {
                \"nome_cliente\": \"$nome_cliente\",
                \"source\": \"whatsapp\",
                \"link\": \"https://example.com/continue\"
            }
        }"

    echo ""
    echo -e "${GREEN}✅ Event posted${NC}"
    echo ""
}

# Function to finish a journey
finish_journey() {
    local journey_id="${1:-onboarding-v2}"
    local customer_number="${2:-5511999999999}"

    echo -e "${BLUE}✅ Finishing journey...${NC}"
    echo "  Journey: $journey_id"
    echo "  Customer: $customer_number"
    echo ""

    curl -X POST "$API_URL/journey/finish" \
        -H "Content-Type: application/json" \
        -d "{
            \"journey_id\": \"$journey_id\",
            \"customer_number\": \"$customer_number\"
        }"

    echo ""
    echo -e "${GREEN}✅ Journey finished${NC}"
    echo ""
}

# Function to inspect customer state
inspect_state() {
    local journey_id="${1:-onboarding-v2}"
    local customer_number="${2:-5511999999999}"
    local key="journey:${journey_id}:${customer_number}:state"

    echo -e "${BLUE}🔍 Inspecting state...${NC}"
    echo "  Key: $key"
    echo ""

    docker exec -it "$REDIS_CONTAINER" redis-cli GET "$key"

    echo ""
}

# Function to inspect recovery history
inspect_history() {
    local journey_id="${1:-onboarding-v2}"
    local customer_number="${2:-5511999999999}"
    local key="journey:${journey_id}:${customer_number}:repiques"

    echo -e "${BLUE}📜 Inspecting recovery history...${NC}"
    echo "  Key: $key"
    echo ""

    docker exec -it "$REDIS_CONTAINER" redis-cli LRANGE "$key" 0 -1

    echo ""
}

# Function to list all journey keys
list_journeys() {
    echo -e "${BLUE}📋 Listing all journeys...${NC}"
    echo ""

    docker exec -it "$REDIS_CONTAINER" redis-cli KEYS "journey:*"

    echo ""
}

# Function to trigger worker manually
trigger_worker() {
    echo -e "${BLUE}🔄 Restarting worker to trigger immediate execution...${NC}"
    echo ""

    cd "$(dirname "$0")/../../docker" || exit 1

    if docker compose version &> /dev/null; then
        docker compose restart worker
    else
        docker-compose restart worker
    fi

    echo ""
    echo -e "${GREEN}✅ Worker restarted. Check logs:${NC}"
    echo "  cd docker && docker compose logs -f worker"
    echo ""
}

# Function to clear all data
clear_all() {
    echo -e "${YELLOW}⚠️  WARNING: This will delete all journey data!${NC}"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo ""

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}🗑️  Clearing all data...${NC}"
        docker exec -it "$REDIS_CONTAINER" redis-cli FLUSHALL
        echo -e "${GREEN}✅ All data cleared${NC}"
    else
        echo "Cancelled."
    fi

    echo ""
}

# Main menu
show_menu() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📱 WhatsApp Recovery System - Test API Helper"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. Post event (default customer)"
    echo "2. Post event (custom)"
    echo "3. Finish journey"
    echo "4. Inspect customer state"
    echo "5. Inspect recovery history"
    echo "6. List all journeys"
    echo "7. Trigger worker manually"
    echo "8. Clear all data"
    echo "9. Exit"
    echo ""
}

# Interactive mode
if [ "$1" == "" ]; then
    while true; do
        show_menu
        read -p "Choose an option: " choice
        echo ""

        case $choice in
            1)
                post_event
                ;;
            2)
                read -p "Journey ID [onboarding-v2]: " journey_id
                read -p "Step [personal-data]: " step
                read -p "Customer Number [5511999999999]: " customer_number
                read -p "Customer Name [João Silva]: " nome_cliente
                post_event "${journey_id:-onboarding-v2}" "${step:-personal-data}" "${customer_number:-5511999999999}" "${nome_cliente:-João Silva}"
                ;;
            3)
                read -p "Journey ID [onboarding-v2]: " journey_id
                read -p "Customer Number [5511999999999]: " customer_number
                finish_journey "${journey_id:-onboarding-v2}" "${customer_number:-5511999999999}"
                ;;
            4)
                read -p "Journey ID [onboarding-v2]: " journey_id
                read -p "Customer Number [5511999999999]: " customer_number
                inspect_state "${journey_id:-onboarding-v2}" "${customer_number:-5511999999999}"
                ;;
            5)
                read -p "Journey ID [onboarding-v2]: " journey_id
                read -p "Customer Number [5511999999999]: " customer_number
                inspect_history "${journey_id:-onboarding-v2}" "${customer_number:-5511999999999}"
                ;;
            6)
                list_journeys
                ;;
            7)
                trigger_worker
                ;;
            8)
                clear_all
                ;;
            9)
                echo "👋 Goodbye!"
                exit 0
                ;;
            *)
                echo -e "${YELLOW}Invalid option${NC}"
                echo ""
                ;;
        esac
    done
else
    # Command line mode
    case "$1" in
        post|event)
            post_event "$2" "$3" "$4" "$5"
            ;;
        finish)
            finish_journey "$2" "$3"
            ;;
        state)
            inspect_state "$2" "$3"
            ;;
        history)
            inspect_history "$2" "$3"
            ;;
        list)
            list_journeys
            ;;
        trigger)
            trigger_worker
            ;;
        clear)
            clear_all
            ;;
        *)
            echo "Usage: $0 {post|finish|state|history|list|trigger|clear} [args...]"
            echo ""
            echo "Or run without arguments for interactive mode."
            exit 1
            ;;
    esac
fi
