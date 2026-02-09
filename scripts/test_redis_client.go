package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/frontandrew/gate/internal/pkg/redis"
)

func main() {
	fmt.Println("=========================================")
	fmt.Println("Redis Client Test")
	fmt.Println("=========================================")
	fmt.Println()

	// Создаем Redis клиент
	client, err := redis.NewClient(redis.Config{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	if err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✅ Connected to Redis")
	fmt.Println()

	ctx := context.Background()

	// Test 1: PING
	fmt.Println("Test 1: PING")
	if err := client.Ping(ctx); err != nil {
		fmt.Printf("❌ PING failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ PING successful")
	fmt.Println()

	// Test 2: SET/GET
	fmt.Println("Test 2: SET/GET")
	testKey := "test:gate:key"
	testValue := "Hello from GATE!"

	if err := client.Set(ctx, testKey, testValue, 1*time.Minute); err != nil {
		fmt.Printf("❌ SET failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ SET %s = %s\n", testKey, testValue)

	value, err := client.Get(ctx, testKey)
	if err != nil {
		fmt.Printf("❌ GET failed: %v\n", err)
		os.Exit(1)
	}
	if value != testValue {
		fmt.Printf("❌ GET returned wrong value: %s\n", value)
		os.Exit(1)
	}
	fmt.Printf("✅ GET %s = %s\n", testKey, value)
	fmt.Println()

	// Test 3: EXISTS
	fmt.Println("Test 3: EXISTS")
	exists, err := client.Exists(ctx, testKey)
	if err != nil {
		fmt.Printf("❌ EXISTS failed: %v\n", err)
		os.Exit(1)
	}
	if exists != 1 {
		fmt.Printf("❌ Key should exist but doesn't\n")
		os.Exit(1)
	}
	fmt.Printf("✅ EXISTS %s = %d\n", testKey, exists)
	fmt.Println()

	// Test 4: INCR (counter)
	fmt.Println("Test 4: INCR (counter)")
	counterKey := "test:gate:counter"

	count1, err := client.Incr(ctx, counterKey)
	if err != nil {
		fmt.Printf("❌ INCR failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ INCR %s = %d\n", counterKey, count1)

	count2, err := client.Incr(ctx, counterKey)
	if err != nil {
		fmt.Printf("❌ INCR failed: %v\n", err)
		os.Exit(1)
	}
	if count2 != count1+1 {
		fmt.Printf("❌ Counter increment failed\n")
		os.Exit(1)
	}
	fmt.Printf("✅ INCR %s = %d\n", counterKey, count2)
	fmt.Println()

	// Test 5: TTL with EXPIRE
	fmt.Println("Test 5: TTL with EXPIRE")
	ttlKey := "test:gate:ttl"

	if err := client.Set(ctx, ttlKey, "temporary", 0); err != nil {
		fmt.Printf("❌ SET failed: %v\n", err)
		os.Exit(1)
	}

	if err := client.Expire(ctx, ttlKey, 10*time.Second); err != nil {
		fmt.Printf("❌ EXPIRE failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ EXPIRE %s = 10s\n", ttlKey)
	fmt.Println()

	// Test 6: DEL
	fmt.Println("Test 6: DEL (cleanup)")
	if err := client.Del(ctx, testKey, counterKey, ttlKey); err != nil {
		fmt.Printf("❌ DEL failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Deleted test keys")
	fmt.Println()

	// Verify deletion
	exists, err = client.Exists(ctx, testKey)
	if err != nil {
		fmt.Printf("❌ EXISTS check failed: %v\n", err)
		os.Exit(1)
	}
	if exists != 0 {
		fmt.Printf("❌ Key should not exist but does\n")
		os.Exit(1)
	}
	fmt.Println("✅ Verified keys deleted")
	fmt.Println()

	fmt.Println("=========================================")
	fmt.Println("✅ All Redis client tests passed!")
	fmt.Println("=========================================")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
