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

	ensureUser(db, "analyst2", "analyst2@example.com", "analyst", "Мария Смирнова", "analyst223")
	ensureUser(db, "analyst3", "analyst3@example.com", "analyst", "Дмитрий Соколов", "analyst323")
	ensureUser(db, "analyst4", "analyst4@example.com", "analyst", "Елена Кузнецова", "analyst423")
}

func ensureUser(db *gorm.DB, username, email, role, fullName, password string) {
	var existing models.User
	if err := db.Where("username = ?", username).First(&existing).Error; err == nil {
		updates := map[string]interface{}{
			"email":     email,
			"role":      role,
			"full_name": fullName,
		}
		db.Model(&existing).Updates(updates)
		return
	}

	user := models.User{
		Username: username,
		Email:    email,
		Role:     role,
		FullName: fullName,
	}
	user.HashPassword(password)
	db.Create(&user)
}
