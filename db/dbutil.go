package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig holds configuration options for database initialization
type DBConfig struct {
	// Path specifies the database file path. Use ":memory:" for in-memory database
	Path string
	// LogLevel specifies the GORM logging level
	LogLevel logger.LogLevel
}

// InitDatabase creates and configures a SQLite database with the given configuration
// The caller is responsible for running migrations after getting the DB instance
func InitDatabase(config DBConfig) (*gorm.DB, error) {
	var dsn string

	if config.Path == ":memory:" {
		dsn = ":memory:"
		slog.Debug("Initializing in-memory database")
	} else {
		// Ensure data directory exists if requested
		dir := filepath.Dir(config.Path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			slog.Error("Failed to create data directory", "dir", dir, "error", err)
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}

		dsn = config.Path
		slog.Debug("Initializing file-based database", "path", config.Path)
	}

	// Open database with specified configuration
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
	})
	if err != nil {
		slog.Error("Failed to connect to database", "dsn", dsn, "error", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure SQLite pragmas
	pragmas := "PRAGMA foreign_keys = ON;"

	// Add performance optimizations for file-based databases
	if config.Path != ":memory:" {
		pragmas += `
		PRAGMA legacy_alter_table = OFF;
		PRAGMA journal_mode       = WAL;
		PRAGMA synchronous        = NORMAL;
		PRAGMA mmap_size          = 134217728;
		PRAGMA journal_size_limit = 27103364;
		PRAGMA cache_size         = 2000;`
	}

	if err := db.Exec(pragmas).Error; err != nil {
		slog.Error("Failed to configure database", "error", err)
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	if config.Path == ":memory:" {
		slog.Debug("Database initialized successfully (in-memory)")
	} else {
		slog.Debug("Database initialized successfully", "path", config.Path)
	}

	return db, nil
}
