package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CreditHistory represents the structure of credit history response
type CreditHistory struct {
	PassportData   string         `json:"passport_data"`
	CreditHistory  string         `json:"credit_history"`
	RiskAssessment RiskAssessment `json:"risk_assessment"`
}

// RiskAssessment contains risk metrics
type RiskAssessment struct {
	Score           float64 `json:"score"`
	RiskLevel       string  `json:"risk_level"` // low, medium, high
	OutstandingDebt float64 `json:"outstanding_debt"`
	PaymentHistory  string  `json:"payment_history"` // good, fair, bad
}

// MockCreditHistoryService simulates an external credit history service
type MockCreditHistoryService struct {
	BaseURL string
	Client  *http.Client
}

// NewMockCreditHistoryService creates a new instance of the mock service
func NewMockCreditHistoryService(baseURL string) *MockCreditHistoryService {
	return &MockCreditHistoryService{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetCreditHistory fetches credit history from the mock service
func (s *MockCreditHistoryService) GetCreditHistory(passportData string) (*CreditHistory, error) {
	// In a real implementation, this would make an HTTP request to an external service
	// For this mock implementation, we'll generate a simulated response

	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// Generate mock credit history based on passport data
	mockHistory := generateMockCreditHistory(passportData)

	return &mockHistory, nil
}

// generateMockCreditHistory generates a mock credit history based on passport data
func generateMockCreditHistory(passportData string) CreditHistory {
	// This is a simplified example - in reality, this would be more sophisticated
	var score float64
	var riskLevel string
	var outstandingDebt float64
	var paymentHistory string

	// Generate different profiles based on passport data
	switch {
	case len(passportData) > 10 && passportData[len(passportData)-1]%2 == 0:
		score = 750 + float64(passportData[len(passportData)-1]%50)
		riskLevel = "low"
		outstandingDebt = float64((passportData[len(passportData)-1] % 10)) * 1000
		paymentHistory = "good"
	case len(passportData) > 10 && passportData[len(passportData)-1]%3 == 0:
		score = 600 + float64(passportData[len(passportData)-1]%100)
		riskLevel = "medium"
		outstandingDebt = float64((passportData[len(passportData)-1]%20)+10) * 1000
		paymentHistory = "fair"
	default:
		score = 500 + float64(passportData[len(passportData)-1]%150)
		riskLevel = "high"
		outstandingDebt = float64((passportData[len(passportData)-1]%30)+20) * 1000
		paymentHistory = "bad"
	}

	return CreditHistory{
		PassportData:  passportData,
		CreditHistory: fmt.Sprintf("Simulated credit history for passport %s", passportData),
		RiskAssessment: RiskAssessment{
			Score:           score,
			RiskLevel:       riskLevel,
			OutstandingDebt: outstandingDebt,
			PaymentHistory:  paymentHistory,
		},
	}
}

// RealCreditHistoryService represents a real implementation that connects to an external service
type RealCreditHistoryService struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewRealCreditHistoryService creates a new instance of the real service
func NewRealCreditHistoryService(baseURL, apiKey string) *RealCreditHistoryService {
	return &RealCreditHistoryService{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetCreditHistory fetches credit history from the real service
func (s *RealCreditHistoryService) GetCreditHistory(passportData string) (*CreditHistory, error) {
	url := fmt.Sprintf("%s/api/v1/credit-history/%s", s.BaseURL, passportData)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var creditHistory CreditHistory
	if err := json.NewDecoder(resp.Body).Decode(&creditHistory); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &creditHistory, nil
}

// SendCreditHistoryUpdate sends an update to the credit history service
func (s *RealCreditHistoryService) SendCreditHistoryUpdate(passportData string, update map[string]interface{}) error {
	url := fmt.Sprintf("%s/api/v1/credit-history/%s", s.BaseURL, passportData)

	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
