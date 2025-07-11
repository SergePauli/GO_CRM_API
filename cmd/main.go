package main

import (
	"GO_CRM_API/internal/config"
	"GO_CRM_API/internal/db"
	"GO_CRM_API/internal/preset"
	"GO_CRM_API/internal/router"
	"log"
	"net/http"

	"fmt"
)

func main() {
	cfg := config.LoadConfig()

	// PostgreSQL
	if err := db.InitPostgres(cfg.PostgresDSN); err != nil {
		log.Fatalf("❌ PostgreSQL init failed: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// Redis
    db.InitRedis(cfg.RedisAddr)

	if err := db.PingRedis(); err != nil {
		log.Fatalf("❌ Redis init failed: %v", err)
	}

	log.Println("✅ Connected to Redis")
    // Initialize all presets
    preset.InitAllPresets()
    fmt.Println("✅ All presets initialized")
    // Initialize routes
    router.InitRoutes()
    // Start HTTP server
    log.Printf("🚀 Starting server on port %s", cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
