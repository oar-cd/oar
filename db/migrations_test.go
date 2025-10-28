package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

// TestMigration0001RenameLastCommitToLocalCommit tests migration 1
func TestMigration0001RenameLastCommitToLocalCommit(t *testing.T) {
	// Create database at migration 0 (before this migration)
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Create schema at migration 0
	err = CreateSchemaAtMigration(db, 0)
	require.NoError(t, err)

	// Insert test data with old column names
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

	// Verify old column exists
	hasLastCommit := db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.True(t, hasLastCommit, "last_commit column should exist before migration")

	// Apply migration 1
	err = RunMigrations(db, 1)
	require.NoError(t, err)

	// Verify last_commit column no longer exists (was renamed)
	hasLastCommit = db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.False(t, hasLastCommit, "last_commit column should not exist after migration")

	// Verify local_commit column exists
	hasLocalCommit := db.Migrator().HasColumn(&ProjectModel{}, "local_commit")
	assert.True(t, hasLocalCommit, "local_commit column should exist after migration")

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

	// Verify migration was recorded
	var migrationCount int64
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "0001_rename_last_commit_to_local_commit").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should be recorded once")

	// Verify idempotency - running again should not fail
	err = RunMigrations(db, 1)
	assert.NoError(t, err, "Migration should be idempotent")

	// Verify migration is still recorded only once
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "0001_rename_last_commit_to_local_commit").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should still be recorded only once")
}

// TestMigration0002RenameWatcherEnabledToAutoDeployEnabled tests migration 2
func TestMigration0002RenameWatcherEnabledToAutoDeployEnabled(t *testing.T) {
	// Create database at migration 1 (before this migration)
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Create schema at migration 1 (has local_commit but still has watcher_enabled)
	err = CreateSchemaAtMigration(db, 1)
	require.NoError(t, err)

	// Insert test data with column names as they exist after migration 1
	testID1 := uuid.New()
	testID2 := uuid.New()

	err = db.Exec(`
		INSERT INTO projects (
			id, name, git_url, git_branch, working_dir, compose_files,
			variables, status, local_commit, watcher_enabled, created_at, updated_at
		) VALUES
			(?, 'project1', 'https://example.com/repo1.git', 'main', '/path1', 'compose.yml', '', 'stopped', 'abc123', 1, datetime('now'), datetime('now')),
			(?, 'project2', 'https://example.com/repo2.git', 'main', '/path2', 'compose.yml', '', 'running', 'def456', 0, datetime('now'), datetime('now'))
	`, testID1, testID2).Error
	require.NoError(t, err)

	// Verify old column exists
	hasWatcherEnabled := db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled")
	assert.True(t, hasWatcherEnabled, "watcher_enabled column should exist before migration")

	// Apply migration 2
	err = RunMigrations(db, 2)
	require.NoError(t, err)

	// Verify watcher_enabled column no longer exists (was renamed)
	hasWatcherEnabled = db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled")
	assert.False(t, hasWatcherEnabled, "watcher_enabled column should not exist after migration")

	// Verify auto_deploy_enabled column exists
	hasAutoDeployEnabled := db.Migrator().HasColumn(&ProjectModel{}, "auto_deploy_enabled")
	assert.True(t, hasAutoDeployEnabled, "auto_deploy_enabled column should exist after migration")

	// Verify data was migrated correctly
	type Result struct {
		ID                string
		AutoDeployEnabled bool
	}

	var results []Result
	err = db.Raw("SELECT id, auto_deploy_enabled FROM projects ORDER BY name").Scan(&results).Error
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check project1 - should have watcher_enabled=1 migrated to auto_deploy_enabled=true
	assert.Equal(t, testID1.String(), results[0].ID)
	assert.True(t, results[0].AutoDeployEnabled)

	// Check project2 - should have watcher_enabled=0 migrated to auto_deploy_enabled=false
	assert.Equal(t, testID2.String(), results[1].ID)
	assert.False(t, results[1].AutoDeployEnabled)

	// Verify migration was recorded
	var migrationCount int64
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "0002_rename_watcher_enabled_to_auto_deploy_enabled").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should be recorded once")

	// Verify idempotency - running again should not fail
	err = RunMigrations(db, 2)
	assert.NoError(t, err, "Migration should be idempotent")

	// Verify migration is still recorded only once
	err = db.Model(&MigrationModel{}).
		Where("name = ?", "0002_rename_watcher_enabled_to_auto_deploy_enabled").
		Count(&migrationCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), migrationCount, "Migration should still be recorded only once")
}

// TestAutoMigrateAllFreshDatabase tests AutoMigrateAll on a fresh database
func TestAutoMigrateAllFreshDatabase(t *testing.T) {
	// Create in-memory database
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Run AutoMigrateAll on fresh database
	err = AutoMigrateAll(db)
	require.NoError(t, err)

	// Verify tables exist with correct schema
	hasProjectsTable := db.Migrator().HasTable(&ProjectModel{})
	assert.True(t, hasProjectsTable, "projects table should exist")

	hasDeploymentsTable := db.Migrator().HasTable(&DeploymentModel{})
	assert.True(t, hasDeploymentsTable, "deployments table should exist")

	// Verify new column names exist
	hasLocalCommit := db.Migrator().HasColumn(&ProjectModel{}, "local_commit")
	assert.True(t, hasLocalCommit, "local_commit column should exist")

	hasAutoDeployEnabled := db.Migrator().HasColumn(&ProjectModel{}, "auto_deploy_enabled")
	assert.True(t, hasAutoDeployEnabled, "auto_deploy_enabled column should exist")

	// Verify old column names do not exist
	hasLastCommit := db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.False(t, hasLastCommit, "last_commit column should not exist")

	hasWatcherEnabled := db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled")
	assert.False(t, hasWatcherEnabled, "watcher_enabled column should not exist")

	// Verify migrations were recorded
	var count int64
	err = db.Model(&MigrationModel{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), count, "Should have 3 migration records")
}

// TestMigration0003CleanupOldMigrationRecords tests migration 3
func TestMigration0003CleanupOldMigrationRecords(t *testing.T) {
	// Create database at migration 2 (before this migration)
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Create schema at migration 2
	err = CreateSchemaAtMigration(db, 2)
	require.NoError(t, err)

	// Manually insert old-style migration records to simulate an upgraded database
	err = db.Create(&MigrationModel{
		Name:      "rename_last_commit_to_local_commit",
		AppliedAt: time.Now(),
	}).Error
	require.NoError(t, err)

	err = db.Create(&MigrationModel{
		Name:      "rename_watcher_enabled_to_auto_deploy_enabled",
		AppliedAt: time.Now(),
	}).Error
	require.NoError(t, err)

	// Verify we have 4 migration records (2 new-style + 2 old-style)
	var countBefore int64
	err = db.Model(&MigrationModel{}).Count(&countBefore).Error
	require.NoError(t, err)
	assert.Equal(t, int64(4), countBefore, "Should have 4 migration records before cleanup")

	// Apply migration 3
	err = RunMigrations(db, 3)
	require.NoError(t, err)

	// Verify old migration records were deleted
	var countAfter int64
	err = db.Model(&MigrationModel{}).Count(&countAfter).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), countAfter, "Should have 3 migration records after cleanup (0001, 0002, 0003)")

	// Verify old-style records are gone
	var oldStyleCount int64
	err = db.Model(&MigrationModel{}).
		Where("name IN ?", []string{
			"rename_last_commit_to_local_commit",
			"rename_watcher_enabled_to_auto_deploy_enabled",
		}).
		Count(&oldStyleCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), oldStyleCount, "Old-style migration records should be deleted")

	// Verify new-style records are still present
	var newStyleCount int64
	err = db.Model(&MigrationModel{}).
		Where("name LIKE ?", "000%").
		Count(&newStyleCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), newStyleCount, "All new-style migration records should be present")
}

// TestIncrementalMigration tests applying all migrations incrementally
func TestIncrementalMigration(t *testing.T) {
	// Create database at migration 0
	db, err := InitDatabase(DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	err = CreateSchemaAtMigration(db, 0)
	require.NoError(t, err)

	// Insert test data at migration 0
	testID := uuid.New()
	err = db.Exec(`
		INSERT INTO projects (
			id, name, git_url, git_branch, working_dir, compose_files,
			variables, status, last_commit, watcher_enabled, created_at, updated_at
		) VALUES (?, 'test-project', 'https://example.com/repo.git', 'main', '/path', 'compose.yml', '', 'stopped', 'abc123', 1, datetime('now'), datetime('now'))
	`, testID).Error
	require.NoError(t, err)

	// Verify initial state (migration 0)
	hasLastCommit := db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.True(t, hasLastCommit, "Should have last_commit at migration 0")

	hasWatcherEnabled := db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled")
	assert.True(t, hasWatcherEnabled, "Should have watcher_enabled at migration 0")

	// Apply migration 1
	err = RunMigrations(db, 1)
	require.NoError(t, err)

	// Verify state after migration 1
	hasLastCommit = db.Migrator().HasColumn(&ProjectModel{}, "last_commit")
	assert.False(t, hasLastCommit, "Should not have last_commit after migration 1")

	hasLocalCommit := db.Migrator().HasColumn(&ProjectModel{}, "local_commit")
	assert.True(t, hasLocalCommit, "Should have local_commit after migration 1")

	hasWatcherEnabled = db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled")
	assert.True(t, hasWatcherEnabled, "Should still have watcher_enabled after migration 1")

	// Apply migration 2
	err = RunMigrations(db, 2)
	require.NoError(t, err)

	// Verify state after migration 2
	hasWatcherEnabled = db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled")
	assert.False(t, hasWatcherEnabled, "Should not have watcher_enabled after migration 2")

	hasAutoDeployEnabled := db.Migrator().HasColumn(&ProjectModel{}, "auto_deploy_enabled")
	assert.True(t, hasAutoDeployEnabled, "Should have auto_deploy_enabled after migration 2")

	// Verify data was preserved through all migrations
	type Result struct {
		ID                string
		LocalCommit       *string
		AutoDeployEnabled bool
	}
	var result Result
	err = db.Raw("SELECT id, local_commit, auto_deploy_enabled FROM projects WHERE id = ?", testID.String()).
		Scan(&result).
		Error
	require.NoError(t, err)
	require.NotNil(t, result.LocalCommit)
	assert.Equal(t, "abc123", *result.LocalCommit, "Data should be preserved")
	assert.True(t, result.AutoDeployEnabled, "watcher_enabled=1 should become auto_deploy_enabled=true")

	// Apply migration 3
	err = RunMigrations(db, 3)
	require.NoError(t, err)

	// Verify all migrations were recorded
	var migrationCount int64
	err = db.Model(&MigrationModel{}).Count(&migrationCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), migrationCount, "Should have 3 migration records")
}
