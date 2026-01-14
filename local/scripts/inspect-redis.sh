#!/bin/bash

# Interactive Redis inspection script

REDIS_CONTAINER="recovery-redis"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check if Redis container is running
if ! docker ps | grep -q "$REDIS_CONTAINER"; then
    echo -e "${RED}❌ Redis container is not running${NC}"
    echo "Start it with: cd docker && docker-compose up -d redis"
    exit 1
fi

# Function to list all journey states
list_states() {
    echo -e "${BLUE}📋 All Journey States${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    docker exec "$REDIS_CONTAINER" redis-cli KEYS "journey:*:*:state" | while read -r key; do
        if [ -n "$key" ]; then
            echo -e "${GREEN}🔑 $key${NC}"
            docker exec "$REDIS_CONTAINER" redis-cli GET "$key" | jq '.' 2>/dev/null || docker exec "$REDIS_CONTAINER" redis-cli GET "$key"
            echo ""
        fi
    done
}

# Function to list all recovery histories
list_histories() {
    echo -e "${BLUE}📜 All Recovery Histories${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    docker exec "$REDIS_CONTAINER" redis-cli KEYS "journey:*:*:repiques" | while read -r key; do
        if [ -n "$key" ]; then
            echo -e "${GREEN}🔑 $key${NC}"
            docker exec "$REDIS_CONTAINER" redis-cli LRANGE "$key" 0 -1 | while read -r entry; do
                if [ -n "$entry" ]; then
                    echo "$entry" | jq '.' 2>/dev/null || echo "$entry"
                fi
            done
            echo ""
        fi
    done
}

# Function to show specific customer
show_customer() {
    local journey_id="$1"
    local customer_number="$2"

    echo -e "${BLUE}👤 Customer: $customer_number${NC}"
    echo -e "${BLUE}🗺️  Journey: $journey_id${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Show state
    local state_key="journey:${journey_id}:${customer_number}:state"
    echo -e "${YELLOW}📊 Current State:${NC}"
    local state=$(docker exec "$REDIS_CONTAINER" redis-cli GET "$state_key")
    if [ -n "$state" ]; then
        echo "$state" | jq '.' 2>/dev/null || echo "$state"
    else
        echo "  No state found"
    fi
    echo ""

    # Show history
    local history_key="journey:${journey_id}:${customer_number}:repiques"
    echo -e "${YELLOW}📜 Recovery History:${NC}"
    local count=$(docker exec "$REDIS_CONTAINER" redis-cli LLEN "$history_key")
    if [ "$count" -gt 0 ]; then
        docker exec "$REDIS_CONTAINER" redis-cli LRANGE "$history_key" 0 -1 | while read -r entry; do
            if [ -n "$entry" ]; then
                echo "$entry" | jq '.' 2>/dev/null || echo "$entry"
                echo ""
            fi
        done
    else
        echo "  No recovery attempts yet"
    fi
    echo ""
}

# Function to show statistics
show_stats() {
    echo -e "${BLUE}📊 Redis Statistics${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    local state_count=$(docker exec "$REDIS_CONTAINER" redis-cli KEYS "journey:*:*:state" | wc -l)
    local history_count=$(docker exec "$REDIS_CONTAINER" redis-cli KEYS "journey:*:*:repiques" | wc -l)
    local total_keys=$(docker exec "$REDIS_CONTAINER" redis-cli DBSIZE | grep -oP '\d+')

    echo "  Active Journey States:     $state_count"
    echo "  Recovery Histories:        $history_count"
    echo "  Total Keys in Database:    $total_keys"
    echo ""

    # Memory usage
    echo -e "${YELLOW}💾 Memory Usage:${NC}"
    docker exec "$REDIS_CONTAINER" redis-cli INFO memory | grep "used_memory_human"
    echo ""
}

# Function to clear all data with confirmation
clear_all() {
    echo -e "${YELLOW}⚠️  WARNING: This will delete ALL data in Redis!${NC}"
    echo ""

    read -p "Are you sure you want to continue? (type 'yes' to confirm): " confirmation

    if [ "$confirmation" == "yes" ]; then
        echo ""
        echo -e "${BLUE}🗑️  Clearing all data...${NC}"
        docker exec "$REDIS_CONTAINER" redis-cli FLUSHALL
        echo -e "${GREEN}✅ All data cleared${NC}"
    else
        echo "Cancelled."
    fi
    echo ""
}

# Function to open Redis CLI
open_cli() {
    echo -e "${BLUE}🔧 Opening Redis CLI...${NC}"
    echo "Type 'exit' or press Ctrl+D to return"
    echo ""
    docker exec -it "$REDIS_CONTAINER" redis-cli
}

# Main menu
show_menu() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔍 Redis Inspector - WhatsApp Recovery System"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. List all journey states"
    echo "2. List all recovery histories"
    echo "3. Show specific customer"
    echo "4. Show statistics"
    echo "5. Clear all data"
    echo "6. Open Redis CLI"
    echo "7. Exit"
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
                list_states
                ;;
            2)
                list_histories
                ;;
            3)
                read -p "Journey ID [onboarding-v2]: " journey_id
                read -p "Customer Number [5511999999999]: " customer_number
                show_customer "${journey_id:-onboarding-v2}" "${customer_number:-5511999999999}"
                ;;
            4)
                show_stats
                ;;
            5)
                clear_all
                ;;
            6)
                open_cli
                ;;
            7)
                echo "👋 Goodbye!"
                exit 0
                ;;
            *)
                echo -e "${YELLOW}Invalid option${NC}"
                ;;
        esac
    done
else
    # Command line mode
    case "$1" in
        states)
            list_states
            ;;
        histories)
            list_histories
            ;;
        customer)
            show_customer "$2" "$3"
            ;;
        stats)
            show_stats
            ;;
        clear)
            clear_all
            ;;
        cli)
            open_cli
            ;;
        *)
            echo "Usage: $0 {states|histories|customer|stats|clear|cli} [args...]"
            echo ""
            echo "Or run without arguments for interactive mode."
            exit 1
            ;;
    esac
fi
