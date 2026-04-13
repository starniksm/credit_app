package handlers

import (
	"fmt"
	"net/http"

	"credit_app/models"
	"credit_app/services"
	"credit_app/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApplicationHandler struct {
	service *services.ApplicationService
}

func NewApplicationHandler(db *gorm.DB) *ApplicationHandler {
	return &ApplicationHandler{
		service: services.NewApplicationService(db),
	}
}

func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	var application models.CreditApplication
	if err := c.ShouldBindJSON(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Received application data: %+v\n", application)

	// Validate required fields
	if application.ClientName == "" && application.FullName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client name is required"})
		return
	}
	if application.RequestedAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Credit amount must be greater than 0"})
		return
	}

	// Set default values if not provided
	if application.CreditType == "" {
		application.CreditType = "ПК" // Default to consumer loan
	}
	if application.Priority == "" {
		application.Priority = "medium"
	}
	if application.Status == "" {
		application.Status = "new"
	}

	// Add action to history for creation
	application.AddActionToHistory("created", "Клиент", "Создание заявки")

	result, err := h.service.CreateApplication(&application)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *ApplicationHandler) GetApplicationByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	application, err := h.service.GetApplicationByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add credit type text for display
	creditTypeText := utils.GetCreditTypeText(application.CreditType)

	// Return with additional field for display
	response := gin.H{
		"application":      application,
		"credit_type_text": creditTypeText,
	}

	c.JSON(http.StatusOK, response)
}

func (h *ApplicationHandler) GetApplications(c *gin.Context) {
	var filter models.ApplicationFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applications, err := h.service.GetApplications(&filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, applications)
}

func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var application models.CreditApplication
	if err := c.ShouldBindJSON(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateApplication(id, &application)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	err := h.service.DeleteApplication(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Application deleted successfully"})
}

func (h *ApplicationHandler) RequestBKIData(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	// Update the application with BKI data
	updatedApplication, err := h.service.RequestBKIData(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "BKI data requested successfully",
		"application": updatedApplication,
	})
}
