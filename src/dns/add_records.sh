#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'


API_URL="http://localhost:6969/add"
BASE_NAMES=("free.local" "db.local" "cache.local" "api.local")
BASE_IPS=("10.1.0.15" "10.1.0.16" "10.1.0.17" "10.1.0.18")
TTL=360
MAX=10  

for i in "${!BASE_NAMES[@]}"; do
    base_name=${BASE_NAMES[$i]}
    ip=${BASE_IPS[$i]}
    for n in $(seq 1 $MAX); do
        name="${base_name%.*}$n.${base_name#*.}"
        curl -s -X POST "$API_URL" \
            -H "Content-Type: application/json" \
            -d "{\"name\": \"$name\", \"ip\": \"$ip\", \"ttl\": $TTL}" \
            && echo -e "${GREEN}Added record: $name -> $ip (TTL $TTL)${NC}" \
            || echo -e "${RED}Failed to add record: $name${NC}"

        sleep 0.2
    done
done

echo -e "${CYAN}Finished adding all records${NC}"
