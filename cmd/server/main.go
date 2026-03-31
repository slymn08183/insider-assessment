package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/slymn08183/insider-assessment/internal/cache"
	"github.com/slymn08183/insider-assessment/internal/config"
	"github.com/slymn08183/insider-assessment/internal/database"
	"github.com/slymn08183/insider-assessment/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if cfg.AutoMigrate {
		if err := database.Migrate(db); err != nil {
			log.Fatal("Failed to migrate:", err)
		}
	}

	rdb, err := cache.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to redis:", err)
	}
	defer rdb.Close()

	_ = db
	_ = rdb

	r := gin.Default()

	healthHandler := &handler.HealthHandler{}
	eventHandler := &handler.EventHandler{}
	metricsHandler := &handler.MetricsHandler{}

	healthHandler.RegisterRoutes(r)
	eventHandler.RegisterRoutes(r)
	metricsHandler.RegisterRoutes(r)

	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
