package db

import (
	"time"

	"gorm.io/gorm"
)

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
	// Run manual migrations BEFORE AutoMigrate to avoid conflicts
	// (e.g., AutoMigrate would create local_commit if we haven't renamed last_commit yet)

	// First, ensure migrations table exists
	if err := db.AutoMigrate(&MigrationModel{}); err != nil {
		return err
	}

	// Run manual migrations
	if err := migrateLastCommitToLocalCommit(db); err != nil {
		return err
	}

	// Now run AutoMigrate for all models
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return err
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

// migrateLastCommitToLocalCommit handles the rename from last_commit to local_commit
func migrateLastCommitToLocalCommit(db *gorm.DB) error {
	migrationName := "rename_last_commit_to_local_commit"

	// Check if migration has already been applied
	applied, err := migrationApplied(db, migrationName)
	if err != nil {
		return err
	}
	if applied {
		return nil // Migration already done
	}

	// Check if old column exists - if not, this is a fresh database and nothing to migrate
	if !db.Migrator().HasColumn(&ProjectModel{}, "last_commit") {
		// Record migration as applied anyway to prevent future attempts
		return recordMigration(db, migrationName)
	}

	// Rename column directly (requires SQLite 3.25.0+)
	if err := db.Exec("ALTER TABLE projects RENAME COLUMN last_commit TO local_commit").Error; err != nil {
		return err
	}

	// Record that migration was applied
	return recordMigration(db, migrationName)
}
