package db

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

func TestGetGormLogLevel(t *testing.T) {
	tests := []struct {
		name           string
		logLevel       slog.Level
		expectedResult logger.LogLevel
	}{
		{
			name:           "debug level returns info",
			logLevel:       slog.LevelDebug,
			expectedResult: logger.Info,
		},
		{
			name:           "info level returns warn",
			logLevel:       slog.LevelInfo,
			expectedResult: logger.Warn,
		},
		{
			name:           "warn level returns warn",
			logLevel:       slog.LevelWarn,
			expectedResult: logger.Warn,
		},
		{
			name:           "error level returns error",
			logLevel:       slog.LevelError,
			expectedResult: logger.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a logger with the specified level
			handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: tt.logLevel,
			})
			originalLogger := slog.Default()
			slog.SetDefault(slog.New(handler))
			defer slog.SetDefault(originalLogger)

			result := getGormLogLevel()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestInitDB(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := InitDB(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify database file was created
	dbPath := tmpDir + "/oar.db"
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Verify we can perform basic operations
	err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)").Error
	assert.NoError(t, err)

	// Test that we can insert and query data
	err = db.Exec("INSERT INTO test_table (id) VALUES (1)").Error
	assert.NoError(t, err)

	var count int64
	err = db.Raw("SELECT COUNT(*) FROM test_table").Scan(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestInitDB_InvalidDirectory(t *testing.T) {
	// Try to initialize database in invalid directory
	invalidDir := "/root/nonexistent"

	db, err := InitDB(invalidDir)
	assert.Error(t, err)
	assert.Nil(t, db)
}

// Test helper function to verify the getGormLogLevel logic in isolation
func TestGetGormLogLevel_DirectCheck(t *testing.T) {
	// Save original logger
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	// Test with debug level
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(debugHandler))

	// Verify that debug logging is enabled
	assert.True(t, slog.Default().Enabled(context.TODO(), slog.LevelDebug))

	result := getGormLogLevel()
	assert.Equal(t, logger.Info, result)

	// Test with error level (highest level, disables lower levels)
	errorHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	slog.SetDefault(slog.New(errorHandler))

	// Verify that debug/info/warn logging is disabled
	assert.False(t, slog.Default().Enabled(context.TODO(), slog.LevelDebug))
	assert.False(t, slog.Default().Enabled(context.TODO(), slog.LevelInfo))
	assert.False(t, slog.Default().Enabled(context.TODO(), slog.LevelWarn))
	assert.True(t, slog.Default().Enabled(context.TODO(), slog.LevelError))

	result = getGormLogLevel()
	assert.Equal(t, logger.Error, result)
}
