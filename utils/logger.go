package utils

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"
)

// LoggerToFile returns a logger interface that writes to a file
func LoggerToFile() logger.Interface {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)
	return newLogger
}

// SetupGinLogger sets up the Gin logger middleware
func SetupGinLogger() gin.HandlerFunc {
	file, _ := os.Create("gin.log")
	gin.DisableConsoleColor()
	return gin.LoggerWithWriter(file)
}
