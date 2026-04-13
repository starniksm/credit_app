package main

import (
	"log"

	"credit_app/config"
	"credit_app/database"
	"credit_app/routes"
	"credit_app/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup logging
	r.Use(utils.SetupGinLogger())

	// Initialize database connection with logging
	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set GORM logger
	db.Logger = utils.LoggerToFile()

	// Run migrations
	database.MigrateTables(db)

	// Create test data
	utils.CreateTestData(db)

	// Setup routes
	routes.SetupRoutes(r, db)

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server error:", err)
	}
}
