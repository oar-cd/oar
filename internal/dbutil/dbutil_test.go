package dbutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

func TestInitDatabase_InMemory(t *testing.T) {
	config := DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	}

	db, err := InitDatabase(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify we can perform basic operations
	err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)").Error
	assert.NoError(t, err)

	// Verify foreign keys are enabled
	var foreignKeysEnabled int
	err = db.Raw("PRAGMA foreign_keys").Scan(&foreignKeysEnabled).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, foreignKeysEnabled)
}

func TestInitDatabase_FileBased(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DBConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := InitDatabase(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Verify we can perform basic operations
	err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)").Error
	assert.NoError(t, err)

	// Verify SQLite pragmas are set correctly
	var journalMode string
	err = db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error
	assert.NoError(t, err)
	assert.Equal(t, "wal", journalMode)

	var foreignKeysEnabled int
	err = db.Raw("PRAGMA foreign_keys").Scan(&foreignKeysEnabled).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, foreignKeysEnabled)
}

func TestInitDatabase_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "dir", "test.db")

	config := DBConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := InitDatabase(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify nested directories were created
	_, err = os.Stat(filepath.Dir(dbPath))
	assert.NoError(t, err)

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestInitDatabase_InvalidDirectory(t *testing.T) {
	// Try to create a database in a path that can't be created
	invalidPath := "/root/nonexistent/test.db"

	config := DBConfig{
		Path:     invalidPath,
		LogLevel: logger.Silent,
	}

	db, err := InitDatabase(config)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to create data directory")
}

func TestDBConfig_LogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	logLevels := []logger.LogLevel{
		logger.Silent,
		logger.Error,
		logger.Warn,
		logger.Info,
	}

	logLevelNames := []string{"Silent", "Error", "Warn", "Info"}

	for i, logLevel := range logLevels {
		t.Run(logLevelNames[i], func(t *testing.T) {
			dbPath := filepath.Join(tmpDir, "test_"+logLevelNames[i]+".db")

			config := DBConfig{
				Path:     dbPath,
				LogLevel: logLevel,
			}

			db, err := InitDatabase(config)
			require.NoError(t, err)
			require.NotNil(t, db)

			// Basic functionality test
			err = db.Exec("CREATE TABLE test (id INTEGER)").Error
			assert.NoError(t, err)
		})
	}
}

func TestInitDatabase_MultipleConnections(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "shared.db")

	config := DBConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	// Create first connection
	db1, err := InitDatabase(config)
	require.NoError(t, err)
	require.NotNil(t, db1)

	// Create table with first connection
	err = db1.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)").Error
	require.NoError(t, err)

	// Insert data with first connection
	err = db1.Exec("INSERT INTO test (value) VALUES (?)", "test1").Error
	require.NoError(t, err)

	// Create second connection to same database
	db2, err := InitDatabase(config)
	require.NoError(t, err)
	require.NotNil(t, db2)

	// Verify second connection can read data from first
	var count int64
	err = db2.Raw("SELECT COUNT(*) FROM test").Scan(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
