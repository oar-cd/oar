// Package db provides functions to initialize and manage the SQLite database for Oar.
package db

import (
	"log/slog"
	"path/filepath"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ch00k/oar/internal/dbutil"
	"github.com/ch00k/oar/models"
)

func InitDB(dataDir string) (*gorm.DB, error) {
	slog.Info("Initializing database", "data_dir", dataDir)
	dbPath := filepath.Join(dataDir, "oar.db")

	// Initialize database using shared utility
	db, err := dbutil.InitDatabase(dbutil.DBConfig{
		Path:     dbPath,
		LogLevel: logger.Info,
	})
	if err != nil {
		return nil, err
	}

	// Run migrations for all models (single source of truth)
	if err := models.AutoMigrateAll(db); err != nil {
		return nil, err
	}

	slog.Info("Database initialized successfully", "path", dbPath)
	return db, nil
}
