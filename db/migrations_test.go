package db

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

func TestMigrateLastCommitToLocalCommit(t *testing.T) {
	// Create in-memory database
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Create migrations table first (simulating AutoMigrate)
	err = db.AutoMigrate(&MigrationModel{})
	require.NoError(t, err)

	// Create old schema with last_commit column
	err = db.Exec(`
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			git_url TEXT NOT NULL,
			git_branch TEXT NOT NULL,
			git_auth_type TEXT,
			git_auth_credentials TEXT,
			working_dir TEXT NOT NULL,
			compose_files TEXT NOT NULL,
			compose_override TEXT,
			variables TEXT NOT NULL,
			status TEXT NOT NULL,
			last_commit TEXT,
			watcher_enabled INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			CHECK(name <> ''),
			CHECK(git_url <> ''),
			CHECK(git_branch <> ''),
			CHECK(working_dir <> ''),
			CHECK(compose_files <> ''),
			CHECK(status <> '')
		)
	`).Error
	require.NoError(t, err)

	// Insert test data with last_commit values
	testID1 := uuid.New()
	testID2 := uuid.New()
	testID3 := uuid.New()

	err = db.Exec(`
		INSERT INTO projects (
			id, name, git_url, git_branch, working_dir, compose_files,
			variables, status, last_commit, watcher_enabled, created_at, updated_at
		) VALUES
			(?, 'project1', 'https://example.com/repo1.git', 'main', '/path1', 'compose.yml', '', 'stopped', 'abc123', 1, datetime('now'), datetime('now')),
			(?, 'project2', 'https://example.com/repo2.git', 'main', '/path2', 'compose.yml', '', 'running', 'def456', 0, datetime('now'), datetime('now')),
			(?, 'project3', 'https://example.com/repo3.git', 'main', '/path3', 'compose.yml', '', 'stopped', NULL, 1, datetime('now'), datetime('now'))
	`, testID1, testID2, testID3).Error
	require.NoError(t, err)

	// Verify old data exists in last_commit column
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM projects WHERE last_commit IS NOT NULL").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "Should have 2 projects with last_commit values")

	// Run the migration (which will rename last_commit to local_commit)
	err = migrateLastCommitToLocalCommit(db)
	require.NoError(t, err)

	// Verify last_commit column no longer exists (was renamed)
	hasLastCommit := db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.False(t, hasLastCommit, "last_commit column should not exist after rename")

	// Verify local_commit column exists
	hasLocalCommit := db.Migrator().HasColumn(&ProjectModel{}, "local_commit")
	assert.True(t, hasLocalCommit, "local_commit column should exist")

	// Verify data was migrated correctly
	type Result struct {
		ID          string
		LocalCommit *string
	}

	var results []Result
	err = db.Raw("SELECT id, local_commit FROM projects ORDER BY name").Scan(&results).Error
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Check project1 - should have migrated commit
	assert.Equal(t, testID1.String(), results[0].ID)
	require.NotNil(t, results[0].LocalCommit)
	assert.Equal(t, "abc123", *results[0].LocalCommit)

	// Check project2 - should have migrated commit
	assert.Equal(t, testID2.String(), results[1].ID)
	require.NotNil(t, results[1].LocalCommit)
	assert.Equal(t, "def456", *results[1].LocalCommit)

	// Check project3 - should remain NULL
	assert.Equal(t, testID3.String(), results[2].ID)
	assert.Nil(t, results[2].LocalCommit)

	// Verify migration was recorded in migrations table
	var migrationCount int64
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "rename_last_commit_to_local_commit").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should be recorded once")

	// Verify migration is idempotent - running again should not fail and should not duplicate
	err = migrateLastCommitToLocalCommit(db)
	assert.NoError(t, err, "Migration should be idempotent")

	// Verify migration is still recorded only once
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "rename_last_commit_to_local_commit").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should still be recorded only once")
}

// TestAutoMigrateAllWithOldSchema tests the real upgrade scenario where
// a database with old schema (last_commit) goes through AutoMigrateAll
func TestAutoMigrateAllWithOldSchema(t *testing.T) {
	// Create in-memory database
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Create migrations table and other tables using GORM (to get exact schema GORM expects)
	err = db.AutoMigrate(&MigrationModel{}, &ProjectModel{}, &DeploymentModel{})
	require.NoError(t, err)

	// Now rename local_commit back to last_commit to simulate old schema
	err = db.Exec("ALTER TABLE projects RENAME COLUMN local_commit TO last_commit").Error
	require.NoError(t, err)

	// Insert test data
	testID := uuid.New()
	err = db.Exec(`
		INSERT INTO projects (
			id, name, git_url, git_branch, working_dir, compose_files,
			variables, status, last_commit, watcher_enabled, created_at, updated_at
		) VALUES (?, 'test-project', 'https://example.com/repo.git', 'main', '/path', 'compose.yml', '', 'stopped', 'abc123', 1, datetime('now'), datetime('now'))
	`, testID).Error
	require.NoError(t, err)

	// Run AutoMigrateAll (this is what happens on app startup with old database)
	// This should: 1) create migrations table, 2) run migration to rename column, 3) run AutoMigrate
	err = AutoMigrateAll(db)
	require.NoError(t, err, "AutoMigrateAll should handle old schema gracefully")

	// Verify last_commit column no longer exists
	hasLastCommit := db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.False(t, hasLastCommit, "last_commit column should not exist after migration")

	// Verify local_commit column exists
	hasLocalCommit := db.Migrator().HasColumn(&ProjectModel{}, "local_commit")
	assert.True(t, hasLocalCommit, "local_commit column should exist after migration")

	// Verify data was preserved
	type Result struct {
		ID          string
		LocalCommit *string
	}
	var result Result
	err = db.Raw("SELECT id, local_commit FROM projects WHERE id = ?", testID.String()).Scan(&result).Error
	require.NoError(t, err)
	require.NotNil(t, result.LocalCommit)
	assert.Equal(t, "abc123", *result.LocalCommit, "Data should be preserved during migration")

	// Verify migration was recorded
	var migrationCount int64
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "rename_last_commit_to_local_commit").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should be recorded")
}
