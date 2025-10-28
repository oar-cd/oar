package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Migration represents a single database migration
type Migration struct {
	ID   int
	Name string
	Up   func(*gorm.DB) error
}

// allMigrations is the ordered list of all migrations
// Each migration has a unique ID and is applied in order
var allMigrations = []Migration{
	{
		ID:   1,
		Name: "0001_rename_last_commit_to_local_commit",
		Up:   migration0001RenameLastCommitToLocalCommit,
	},
	{
		ID:   2,
		Name: "0002_rename_watcher_enabled_to_auto_deploy_enabled",
		Up:   migration0002RenameWatcherEnabledToAutoDeployEnabled,
	},
	{
		ID:   3,
		Name: "0003_cleanup_old_migration_records",
		Up:   migration0003CleanupOldMigrationRecords,
	},
}

// AllModels returns all the models that need to be migrated
// This is the single source of truth for database migrations
func AllModels() []any {
	return []any{
		&MigrationModel{},
		&ProjectModel{},
		&DeploymentModel{},
	}
}

// AutoMigrateAll runs auto-migration for all application models
func AutoMigrateAll(db *gorm.DB) error {
	// First, ensure migrations table exists
	if err := db.AutoMigrate(&MigrationModel{}); err != nil {
		return err
	}

	// Run all manual migrations in order
	if err := RunMigrations(db, len(allMigrations)); err != nil {
		return err
	}

	// Now run AutoMigrate for all models
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return err
	}

	return nil
}

// RunMigrations runs all migrations up to and including the specified ID
// If targetID is 0 or negative, all migrations are run
func RunMigrations(db *gorm.DB, targetID int) error {
	if targetID <= 0 {
		targetID = len(allMigrations)
	}

	for _, migration := range allMigrations {
		if migration.ID > targetID {
			break
		}

		// Check if migration has already been applied
		applied, err := migrationApplied(db, migration.Name)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", migration.Name, err)
		}
		if applied {
			continue
		}

		// Run the migration
		if err := migration.Up(db); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}

		// Record that migration was applied
		if err := recordMigration(db, migration.Name); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
		}
	}

	return nil
}

// migrationApplied checks if a migration has already been applied
func migrationApplied(db *gorm.DB, name string) (bool, error) {
	var count int64
	err := db.Model(&MigrationModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// recordMigration records that a migration has been applied
func recordMigration(db *gorm.DB, name string) error {
	migration := MigrationModel{
		Name:      name,
		AppliedAt: time.Now(),
	}
	return db.Create(&migration).Error
}

// Schema snapshots represent the database state at each migration point
// These are used by tests to create databases at specific migration versions

// CreateSchemaAtMigration creates the database schema as it existed at a specific migration version
// migrationID 0 = initial schema before any migrations
// migrationID N = schema after applying migrations 1 through N
func CreateSchemaAtMigration(db *gorm.DB, migrationID int) error {
	// First ensure migrations table exists
	if err := db.AutoMigrate(&MigrationModel{}); err != nil {
		return err
	}

	// Create initial schema (before any migrations)
	if err := createInitialSchema(db); err != nil {
		return err
	}

	// Apply migrations up to the target
	if migrationID > 0 {
		return RunMigrations(db, migrationID)
	}

	return nil
}

// createInitialSchema creates the schema as it existed before any migrations (migration 0)
func createInitialSchema(db *gorm.DB) error {
	// Create projects table with original column names
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
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
			remote_commit TEXT,
			watcher_enabled INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
}

// migration0001RenameLastCommitToLocalCommit handles the rename from last_commit to local_commit
func migration0001RenameLastCommitToLocalCommit(db *gorm.DB) error {
	// Check if old column exists - if not, this is a fresh database and nothing to migrate
	if !db.Migrator().HasColumn(&ProjectModel{}, "last_commit") {
		return nil // Nothing to migrate
	}

	// Rename column directly (requires SQLite 3.25.0+)
	return db.Exec("ALTER TABLE projects RENAME COLUMN last_commit TO local_commit").Error
}

// migration0002RenameWatcherEnabledToAutoDeployEnabled handles the rename from watcher_enabled to auto_deploy_enabled
func migration0002RenameWatcherEnabledToAutoDeployEnabled(db *gorm.DB) error {
	// Check if old column exists - if not, this is a fresh database and nothing to migrate
	if !db.Migrator().HasColumn(&ProjectModel{}, "watcher_enabled") {
		return nil // Nothing to migrate
	}

	// Rename column directly (requires SQLite 3.25.0+)
	return db.Exec("ALTER TABLE projects RENAME COLUMN watcher_enabled TO auto_deploy_enabled").Error
}

// migration0003CleanupOldMigrationRecords removes duplicate migration records from the old naming scheme
func migration0003CleanupOldMigrationRecords(db *gorm.DB) error {
	// Delete old migration records that used the old naming convention (without number prefix)
	oldMigrationNames := []string{
		"rename_last_commit_to_local_commit",
		"rename_watcher_enabled_to_auto_deploy_enabled",
	}

	for _, name := range oldMigrationNames {
		if err := db.Where("name = ?", name).Delete(&MigrationModel{}).Error; err != nil {
			return err
		}
	}

	return nil
}
