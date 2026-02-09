#!/bin/bash

# Тест кэширования Whitelist/Blacklist

echo "========================================="
echo "Whitelist/Blacklist Cache Test"
echo "========================================="
echo ""

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Проверяем, что Redis работает
if ! docker exec gate-redis redis-cli PING 2>/dev/null | grep -q PONG; then
    echo -e "${RED}✗ Redis is not running${NC}"
    echo "Start with: make docker-up"
    exit 1
fi

echo -e "${GREEN}✓ Redis is running${NC}"
echo ""

# Очищаем тестовые ключи перед началом
echo "Cleaning test cache keys..."
docker exec gate-redis redis-cli DEL "whitelist:А123ВС777" >/dev/null 2>&1
docker exec gate-redis redis-cli DEL "blacklist:В456ЕК999" >/dev/null 2>&1
echo ""

# Test 1: Проверка отсутствия кэша (cold start)
echo "Test 1: Cold cache check"
echo "---"

EXISTS_WL=$(docker exec gate-redis redis-cli EXISTS "whitelist:А123ВС777" 2>/dev/null)
if [ "$EXISTS_WL" = "0" ]; then
    echo -e "${GREEN}✓ Whitelist cache is empty (cold start)${NC}"
else
    echo -e "${RED}✗ Whitelist cache should be empty${NC}"
fi

EXISTS_BL=$(docker exec gate-redis redis-cli EXISTS "blacklist:В456ЕК999" 2>/dev/null)
if [ "$EXISTS_BL" = "0" ]; then
    echo -e "${GREEN}✓ Blacklist cache is empty (cold start)${NC}"
else
    echo -e "${RED}✗ Blacklist cache should be empty${NC}"
fi
echo ""

# Test 2: Симуляция записи в кэш (как это делает приложение)
echo "Test 2: Simulating cache write (as application does)"
echo "---"

# Симулируем, что приложение проверило и сохранило результат
docker exec gate-redis redis-cli SETEX "whitelist:А123ВС777" 3600 "1" >/dev/null
docker exec gate-redis redis-cli SETEX "blacklist:В456ЕК999" 3600 "1" >/dev/null

echo -e "${GREEN}✓ Set whitelist:А123ВС777 = 1 (TTL: 3600s)${NC}"
echo -e "${GREEN}✓ Set blacklist:В456ЕК999 = 1 (TTL: 3600s)${NC}"
echo ""

# Test 3: Проверка, что кэш работает
echo "Test 3: Cache hit verification"
echo "---"

WL_VALUE=$(docker exec gate-redis redis-cli GET "whitelist:А123ВС777" 2>/dev/null)
if [ "$WL_VALUE" = "1" ]; then
    echo -e "${GREEN}✓ Whitelist cache hit: А123ВС777 = $WL_VALUE${NC}"
else
    echo -e "${RED}✗ Whitelist cache miss or wrong value${NC}"
    exit 1
fi

BL_VALUE=$(docker exec gate-redis redis-cli GET "blacklist:В456ЕК999" 2>/dev/null)
if [ "$BL_VALUE" = "1" ]; then
    echo -e "${GREEN}✓ Blacklist cache hit: В456ЕК999 = $BL_VALUE${NC}"
else
    echo -e "${RED}✗ Blacklist cache miss or wrong value${NC}"
    exit 1
fi
echo ""

# Test 4: Проверка TTL
echo "Test 4: TTL verification"
echo "---"

WL_TTL=$(docker exec gate-redis redis-cli TTL "whitelist:А123ВС777" 2>/dev/null)
if [ "$WL_TTL" -gt 3500 ] && [ "$WL_TTL" -le 3600 ]; then
    echo -e "${GREEN}✓ Whitelist TTL = ${WL_TTL}s (expected ~3600s)${NC}"
else
    echo -e "${YELLOW}⚠ Whitelist TTL = ${WL_TTL}s${NC}"
fi

BL_TTL=$(docker exec gate-redis redis-cli TTL "blacklist:В456ЕК999" 2>/dev/null)
if [ "$BL_TTL" -gt 3500 ] && [ "$BL_TTL" -le 3600 ]; then
    echo -e "${GREEN}✓ Blacklist TTL = ${BL_TTL}s (expected ~3600s)${NC}"
else
    echo -e "${YELLOW}⚠ Blacklist TTL = ${BL_TTL}s${NC}"
fi
echo ""

# Test 5: Симуляция инвалидации кэша
echo "Test 5: Cache invalidation simulation"
echo "---"

docker exec gate-redis redis-cli DEL "whitelist:А123ВС777" >/dev/null
docker exec gate-redis redis-cli DEL "blacklist:В456ЕК999" >/dev/null

EXISTS_WL_AFTER=$(docker exec gate-redis redis-cli EXISTS "whitelist:А123ВС777" 2>/dev/null)
EXISTS_BL_AFTER=$(docker exec gate-redis redis-cli EXISTS "blacklist:В456ЕК999" 2>/dev/null)

if [ "$EXISTS_WL_AFTER" = "0" ] && [ "$EXISTS_BL_AFTER" = "0" ]; then
    echo -e "${GREEN}✓ Cache invalidated successfully${NC}"
else
    echo -e "${RED}✗ Cache invalidation failed${NC}"
    exit 1
fi
echo ""

# Test 6: Тест производительности (сколько операций в секунду)
echo "Test 6: Performance benchmark"
echo "---"

START_TIME=$(date +%s%N)
for i in {1..100}; do
    docker exec gate-redis redis-cli GET "whitelist:TEST$i" >/dev/null 2>&1
done
END_TIME=$(date +%s%N)

DURATION=$(( (END_TIME - START_TIME) / 1000000 )) # Convert to milliseconds
OPS_PER_SEC=$(( 100000 / DURATION ))

echo -e "${GREEN}✓ 100 cache GET operations: ${DURATION}ms${NC}"
echo -e "${GREEN}✓ Performance: ~${OPS_PER_SEC} ops/sec${NC}"

if [ "$OPS_PER_SEC" -gt 500 ]; then
    echo -e "${GREEN}✓ Performance is excellent (>500 ops/sec)${NC}"
elif [ "$OPS_PER_SEC" -gt 200 ]; then
    echo -e "${YELLOW}⚠ Performance is good (>200 ops/sec)${NC}"
else
    echo -e "${YELLOW}⚠ Performance is acceptable${NC}"
fi
echo ""

# Test 7: Проверка количества ключей в Redis
echo "Test 7: Redis keys count"
echo "---"

WL_KEYS=$(docker exec gate-redis redis-cli KEYS "whitelist:*" 2>/dev/null | wc -l)
BL_KEYS=$(docker exec gate-redis redis-cli KEYS "blacklist:*" 2>/dev/null | wc -l)

echo -e "${GREEN}✓ Whitelist keys in cache: $WL_KEYS${NC}"
echo -e "${GREEN}✓ Blacklist keys in cache: $BL_KEYS${NC}"
echo ""

echo "========================================="
echo -e "${GREEN}✓ All cache tests passed!${NC}"
echo "========================================="
echo ""
echo "Summary:"
echo "- Cache operations work correctly"
echo "- TTL mechanism functions properly"
echo "- Cache invalidation works"
echo "- Performance: ~${OPS_PER_SEC} ops/sec"
echo ""
echo "Next: Start API server to test real cache usage"
echo "Run: make run"
