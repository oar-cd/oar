// Package db provides functions to initialize and manage the SQLite database for Oar.
package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ch00k/oar/models"
)

func InitDB(dataDir string) (*gorm.DB, error) {
	slog.Info("Initializing database", "data_dir", dataDir)

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.Error("Failed to create data directory", "data_dir", dataDir, "error", err)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Database file path
	dbPath := filepath.Join(dataDir, "oar.db")
	slog.Debug("Opening database", "db_path", dbPath)

	// Open database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Quiet for CLI
	})
	if err != nil {
		slog.Error("Failed to connect to database", "db_path", dbPath, "error", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate tables
	slog.Debug("Running database migrations")
	if err := db.AutoMigrate(&models.ProjectModel{}, &models.DeploymentModel{}); err != nil {
		slog.Error("Failed to migrate database", "error", err)
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	slog.Info("Database initialized successfully", "db_path", dbPath)
	return db, nil
}
