#!/bin/bash

# Скрипт для проверки подключения к Redis

echo "========================================="
echo "Redis Connection Test"
echo "========================================="
echo ""

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Проверяем, запущен ли Redis
echo "1. Checking if Redis is running..."
if docker ps | grep -q gate-redis; then
    echo -e "${GREEN}✓ Redis container is running${NC}"
else
    echo -e "${RED}✗ Redis container is not running${NC}"
    echo "Starting Redis with docker-compose..."
    cd "$(dirname "$0")/.." && make docker-up
fi

echo ""
echo "2. Testing Redis connection..."

# Тестируем через redis-cli
if docker exec gate-redis redis-cli PING 2>/dev/null | grep -q PONG; then
    echo -e "${GREEN}✓ Redis PING successful${NC}"
else
    echo -e "${RED}✗ Redis PING failed${NC}"
    exit 1
fi

echo ""
echo "3. Testing basic Redis operations..."

# SET операция
docker exec gate-redis redis-cli SET test:key "Hello Redis" >/dev/null
echo -e "${GREEN}✓ SET test:key${NC}"

# GET операция
VALUE=$(docker exec gate-redis redis-cli GET test:key 2>/dev/null)
if [ "$VALUE" = "Hello Redis" ]; then
    echo -e "${GREEN}✓ GET test:key = '$VALUE'${NC}"
else
    echo -e "${RED}✗ GET test:key failed${NC}"
    exit 1
fi

# DEL операция
docker exec gate-redis redis-cli DEL test:key >/dev/null
echo -e "${GREEN}✓ DEL test:key${NC}"

echo ""
echo "4. Testing Redis with TTL..."

# SET с TTL
docker exec gate-redis redis-cli SETEX test:ttl 5 "Expires in 5 seconds" >/dev/null
TTL=$(docker exec gate-redis redis-cli TTL test:ttl 2>/dev/null)
if [ "$TTL" -gt 0 ] && [ "$TTL" -le 5 ]; then
    echo -e "${GREEN}✓ SETEX with TTL = ${TTL}s${NC}"
else
    echo -e "${RED}✗ SETEX with TTL failed${NC}"
    exit 1
fi

# Cleanup
docker exec gate-redis redis-cli DEL test:ttl >/dev/null

echo ""
echo "5. Redis Info:"
echo "---"
docker exec gate-redis redis-cli INFO server | grep "redis_version\|uptime_in_seconds\|tcp_port"
echo "---"

echo ""
echo -e "${GREEN}========================================="
echo "✓ All Redis tests passed!"
echo "=========================================${NC}"
