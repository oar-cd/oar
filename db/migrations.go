package db

import "gorm.io/gorm"

// AllModels returns all the models that need to be migrated
// This is the single source of truth for database migrations
func AllModels() []any {
	return []any{
		&ProjectModel{},
		&DeploymentModel{},
	}
}

// AutoMigrateAll runs auto-migration for all application models
func AutoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}
