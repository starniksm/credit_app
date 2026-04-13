package database

import (
	"log"

	"credit_app/models"

	"gorm.io/gorm"
)

func MigrateTables(db *gorm.DB) {
	// AutoMigrate will only add missing columns/tables, NOT drop data
	err := db.AutoMigrate(
		&models.CreditApplication{},
		&models.User{},
		&models.Meeting{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database tables:", err)
	}

	// Create default users if not exists
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		// Admin - superuser who can access both roles
		adminUser := models.User{
			Username: "admin",
			Email:    "admin@example.com",
			Role:     "admin",
			FullName: "Администратор",
		}
		adminUser.HashPassword("admin123")
		db.Create(&adminUser)

		// Analyst user
		analystUser := models.User{
			Username: "analyst",
			Email:    "analyst@example.com",
			Role:     "analyst",
			FullName: "Александр Иванов",
		}
		analystUser.HashPassword("analyst123")
		db.Create(&analystUser)

		// Representative user
		repUser := models.User{
			Username: "rep",
			Email:    "rep@example.com",
			Role:     "representative",
			FullName: "Иван Петров",
		}
		repUser.HashPassword("rep123")
		db.Create(&repUser)
	}
}
