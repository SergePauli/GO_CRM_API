package main

import (
	"GO_CRM_API/internal/config"
	"GO_CRM_API/internal/repo"
	"fmt"
	"log"
)

func main() {
	cfg := config.LoadConfig()

	// PostgreSQL
	db, err := repo.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("❌ Postgres error: %v", err)
	}
	defer db.Close()
	log.Println("✅ Connected to PostgreSQL")

	// Redis
	rdb := repo.NewRedis(cfg.RedisAddr)
	if err := repo.PingRedis(rdb); err != nil {
		log.Fatalf("❌ Redis error: %v", err)
	}
	log.Println("✅ Connected to Redis")

	fmt.Println("App is running on port", cfg.Port)
}
