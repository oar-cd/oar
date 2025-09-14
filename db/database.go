// Package db provides functions to initialize and manage the SQLite database for Oar.
package db

import (
	"context"
	"log/slog"
	"path/filepath"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(dataDir string) (*gorm.DB, error) {
	slog.Debug("Initializing database", "data_dir", dataDir)
	dbPath := filepath.Join(dataDir, "oar.db")

	// Set GORM log level based on application log level
	gormLogLevel := getGormLogLevel()

	// Initialize database using shared utility
	db, err := InitDatabase(DBConfig{
		Path:     dbPath,
		LogLevel: gormLogLevel,
	})
	if err != nil {
		return nil, err
	}

	slog.Debug("Database initialized successfully", "path", dbPath)
	return db, nil
}

// getGormLogLevel maps application log level to corresponding GORM log level
func getGormLogLevel() logger.LogLevel {
	// Check the current slog level to determine appropriate GORM level
	ctx := slog.Default()

	if ctx.Enabled(context.TODO(), slog.LevelDebug) {
		return logger.Info // Show SQL queries only when debug logging is enabled
	} else if ctx.Enabled(context.TODO(), slog.LevelInfo) {
		return logger.Warn // Show warnings but not SQL queries
	} else if ctx.Enabled(context.TODO(), slog.LevelWarn) {
		return logger.Warn // Show warnings
	} else if ctx.Enabled(context.TODO(), slog.LevelError) {
		return logger.Error // Show only errors for error-level
	} else {
		return logger.Silent // Silent level - disable all GORM logging
	}
}
