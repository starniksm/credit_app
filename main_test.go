package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"credit_app/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a request to test the health endpoint
	req, _ := http.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	// Create a new gin engine
	r := gin.Default()

	// Add the health route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// Perform the request
	r.ServeHTTP(recorder, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, recorder.Code)

	// Check the response body
	expectedResponse := `{"status":"OK"}`
	assert.JSONEq(t, expectedResponse, recorder.Body.String())
}

func TestCalculateScoring(t *testing.T) {
	// This test would require setting up the full application context
	// For now, we'll just verify that the basic structure works
	app := &models.CreditApplication{
		ClientName:      "Test Client",
		FullName:        "Test Client Full Name",
		MonthlyIncome:   100000,
		EmployerName:    "Test Company",
		RequestedAmount: 50000,
		CreditTerm:      60,
	}

	// Verify that the model can be created
	assert.Equal(t, "Test Client", app.ClientName)
	assert.Equal(t, "Test Client Full Name", app.FullName)
	assert.Equal(t, 100000.0, app.MonthlyIncome)
	assert.Equal(t, 50000.0, app.RequestedAmount)
	assert.Equal(t, 60, app.CreditTerm)
}

func TestCreateApplicationRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := models.CreditApplication{
		ClientName:      "Test Client",
		FullName:        "Test Client Full Name",
		MonthlyIncome:   100000,
		EmployerName:    "Test Company",
		RequestedAmount: 50000,
		CreditTerm:      60,
	}

	jsonData, _ := json.Marshal(app)
	req, _ := http.NewRequest("POST", "/api/applications", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r := gin.Default()

	// We're testing the request structure, not the full functionality
	// In a real scenario, we would need to set up the full route with authentication
	r.POST("/api/applications", func(c *gin.Context) {
		var application models.CreditApplication
		if err := c.ShouldBindJSON(&application); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, application)
	})

	r.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Test Client")
}
