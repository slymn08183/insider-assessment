package database

import (
	"fmt"
	"log"

	"github.com/slymn08183/insider-assessment/internal/config"
	"github.com/slymn08183/insider-assessment/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Database connected")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}
	log.Println("Database migrated")
	return nil
}
