package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slymn08183/insider-assessment/internal/cache"
	"github.com/slymn08183/insider-assessment/internal/config"
	"github.com/slymn08183/insider-assessment/internal/database"
	"github.com/slymn08183/insider-assessment/internal/handler"
	"github.com/slymn08183/insider-assessment/internal/repository"
	"github.com/slymn08183/insider-assessment/internal/worker"
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
	defer func() { _ = sqlDB.Close() }()

	if cfg.AutoMigrate {
		if err := database.Migrate(db); err != nil {
			log.Fatal("Failed to migrate:", err)
		}
	}

	rdb, err := cache.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to redis:", err)
	}
	defer func() { _ = rdb.Close() }()

	queue := cache.NewEventQueue(rdb)
	repo := repository.NewEventRepository(db)

	w := worker.New(queue, repo, cfg.BatchSize, cfg.FlushInterval)
	w.Start()

	r := gin.Default()

	healthHandler := &handler.HealthHandler{}
	eventHandler := handler.NewEventHandler(queue)
	metricsHandler := handler.NewMetricsHandler(repo)

	healthHandler.RegisterRoutes(r)
	eventHandler.RegisterRoutes(r)
	metricsHandler.RegisterRoutes(r)

	// HTTP server — manual setup for graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Start server in separate goroutine
	go func() {
		log.Printf("Server starting on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server failed:", err)
		}
	}()

	// Wait for shutdown signal (Ctrl+C or kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// HTTP server stop accepting, wait for in-flight requests (5s timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Worker flush remaining batch
	w.Stop()

	log.Println("Server exited cleanly")
}
