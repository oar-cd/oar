package app

import (
	"github.com/ch00k/oar/db"
	"github.com/ch00k/oar/services"
	"gorm.io/gorm"
)

var (
	database       *gorm.DB
	projectService *services.ProjectService
)

func Initialize(dataDir string) error {
	var err error
	database, err = db.InitDB(dataDir)
	if err != nil {
		return err
	}

	// Initialize services
	projectService = services.NewProjectService(database)
	return nil
}

func GetDB() *gorm.DB {
	return database
}

func GetProjectService() *services.ProjectService {
	return projectService
}
