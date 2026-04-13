package utils

import (
	"errors"
	"math/rand"
	"time"

	"credit_app/models"
)

// CalculateScoring calculates a scoring value based on application data
func CalculateScoring(app *models.CreditApplication) (float64, error) {
	if app == nil {
		return 0, errors.New("application cannot be nil")
	}

	// Base score calculation
	baseScore := 500.0

	// Adjust based on income (use MonthlyIncome)
	incomeFactor := app.MonthlyIncome / 50000.0 // Normalize to 50k base income
	baseScore += incomeFactor * 100

	// Adjust based on requested amount vs income
	if app.MonthlyIncome > 0 {
		amountToIncomeRatio := app.RequestedAmount / app.MonthlyIncome
		if amountToIncomeRatio > 5 {
			// Too high amount compared to income
			baseScore -= 200
		} else if amountToIncomeRatio > 3 {
			baseScore -= 100
		} else if amountToIncomeRatio > 1 {
			baseScore -= 50
		}
	}

	// Adjust based on credit term
	if app.CreditTerm > 120 { // More than 10 years
		baseScore -= 100
	} else if app.CreditTerm > 60 { // More than 5 years
		baseScore -= 50
	}

	// Adjust based on employer name (could indicate stability)
	if app.EmployerName != "" {
		baseScore += 20
	}

	// Add some randomness to simulate real-world scoring variation
	rand.Seed(time.Now().UnixNano())
	randomAdjustment := rand.Float64()*40 - 20 // Random value between -20 and 20
	baseScore += randomAdjustment

	// Ensure score is within reasonable bounds
	if baseScore < 300 {
		baseScore = 300
	} else if baseScore > 850 {
		baseScore = 850
	}

	return baseScore, nil
}

// DetermineStatusByScore determines the application status based on the scoring value
func DetermineStatusByScore(score float64) string {
	if score >= 700 {
		return "approved"
	} else if score >= 600 {
		return "manual_review"
	} else {
		return "rejected"
	}
}

// FetchCreditHistory simulates fetching credit history from an external service
func FetchCreditHistory(passportData string) (string, error) {
	// This function would normally call an external API
	// For simulation purposes, we'll generate a random credit history

	histories := []string{
		"Good: No late payments, 3 previous loans paid on time",
		"Fair: One late payment over 30 days, 2 previous loans",
		"Good: All payments on time, 5 previous loans paid off",
		"Bad: Multiple late payments, 1 defaulted loan",
		"Good: Long credit history with consistent payments",
		"Fair: Limited credit history, no major issues",
		"Bad: Recent bankruptcy filing",
		"Excellent: Perfect payment history, low utilization",
	}

	rand.Seed(time.Now().UnixNano())
	history := histories[rand.Intn(len(histories))]

	return history, nil
}
