package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type CreditApplication struct {
	ID                     string     `json:"id" gorm:"primaryKey"`
	ClientName             string     `json:"client_name" gorm:"not null"`
	FullName               string     `json:"full_name"`
	BirthDate              *time.Time `json:"birth_date"`
	Age                    int        `json:"age"`
	Gender                 string     `json:"gender"`
	Phone                  string     `json:"phone"`
	Email                  string     `json:"email"`
	PassportSeries         string     `json:"passport_series"`
	PassportNumber         string     `json:"passport_number"`
	PassportIssueDate      *time.Time `json:"passport_issue_date"`
	PassportDepartmentCode string     `json:"passport_department_code"`
	RegistrationAddress    string     `json:"registration_address"`
	ResidenceAddress       string     `json:"residence_address"`
	ResidenceDuration      string     `json:"residence_duration"`
	CreditType             string     `json:"credit_type" gorm:"default:'ПК'"`
	RequestedAmount        float64    `json:"requested_amount" gorm:"not null"`
	CreditTerm             int        `json:"credit_term"`
	CreditPurpose          string     `json:"credit_purpose"`
	MonthlyPayment         float64    `json:"monthly_payment"`
	EmploymentStatus       string     `json:"employment_status"`
	EmployerName           string     `json:"employer_name"`
	Position               string     `json:"position"`
	CurrentJobExperience   string     `json:"current_job_experience"`
	TotalExperience        string     `json:"total_experience"`
	MonthlyIncome          float64    `json:"monthly_income"`
	AdditionalIncome       string     `json:"additional_income"`
	Priority               string     `json:"priority"`
	Status                 string     `json:"status" gorm:"default:new"`
	CardIssued             bool       `json:"card_issued" gorm:"default:false"`
	ReviewStartedAt        *time.Time `json:"review_started_at,omitempty"`
	ReviewerID             string     `json:"reviewer_id,omitempty"`
	AnalystName            string     `json:"analyst_name"`
	Score                  float64    `json:"score,omitempty"`

	// Additional fields needed by the service
	ApprovedAmount     float64                  `json:"approved_amount,omitempty"`
	Expenses           float64                  `json:"expenses,omitempty"`
	RepaymentTerm      int                      `json:"repayment_term,omitempty"`
	EmploymentType     string                   `json:"employment_type,omitempty"`
	WorkExperience     int                      `json:"work_experience,omitempty"`
	EmployerAddress    string                   `json:"employer_address,omitempty"`
	Notes              string                   `json:"notes,omitempty"`
	Documents          []map[string]interface{} `json:"documents,omitempty" gorm:"serializer:json"`
	DecisionReason     string                   `json:"decision_reason,omitempty"`
	ReviewCompletedAt  *time.Time               `json:"review_completed_at,omitempty"`
	LastStatusChangeAt *time.Time               `json:"last_status_change_at,omitempty"`

	// BKI Data
	CreditScore          int     `json:"credit_score"`
	TotalCredits         int     `json:"total_credits"`
	ActiveCredits        int     `json:"active_credits"`
	ClosedCredits        int     `json:"closed_credits"`
	DelayedPayments12m   int     `json:"delayed_payments_12m"`
	CurrentDelinquencies bool    `json:"current_delinquencies"`
	MaxDelinquencyDays   int     `json:"max_delinquency_days"`
	TotalDebt            float64 `json:"total_debt"`
	DTIRatio             float64 `json:"dti_ratio"`
	AvailableCreditLimit float64 `json:"available_credit_limit"`
	ActiveCreditsList    string  `json:"active_credits_list,omitempty"`

	// Scoring and AI fields
	InitialScore         int     `json:"initial_score,omitempty"`
	CreditHistoryScore   int     `json:"credit_history_score"`
	IncomeStabilityScore int     `json:"income_stability_score"`
	DebtBurdenScore      int     `json:"debt_burden_score"`
	AgeFactorScore       int     `json:"age_factor_score"`
	EmploymentScore      int     `json:"employment_score"`
	SolvabilityScore     int     `json:"solvability_score"`
	DebtBurdenRatio      float64 `json:"debt_burden_ratio,omitempty"`
	RiskScore            int     `json:"risk_score"`
	AIRecommendation     string  `json:"ai_recommendation"`
	RecommendationReason string  `json:"recommendation_reason"`
	PositiveFactors      string  `json:"positive_factors,omitempty"`
	RiskFactors          string  `json:"risk_factors,omitempty"`
	AIComment            string  `json:"ai_comment"`
	AIScore              int     `json:"ai_score,omitempty"`
	DelinquencyHistory   string  `json:"delinquency_history,omitempty"`
	DebtBurdenDetails    string  `json:"debt_burden_details,omitempty"`
	FactorsAnalysis      string  `json:"factors_analysis,omitempty"`

	// Additional fields
	CreditHistory string    `json:"credit_history,omitempty"`
	ActionHistory string    `json:"action_history,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ApplicationFilter struct {
	Status     string `form:"status"`
	ClientName string `form:"client_name"`
	Page       int    `form:"page,default=1"`
	Limit      int    `form:"limit,default=10"`

	// Additional fields for filtering
	CreditType string `form:"credit_type"`
	Priority   string `form:"priority"`
	Search     string `form:"search"`
	SortBy     string `form:"sort_by"`
	SortOrder  string `form:"sort_order"`
	Offset     int    `form:"offset"`
}

// TableName specifies the table name for CreditApplication
func (CreditApplication) TableName() string {
	return "credit_applications"
}

// BeforeCreate hook to set default status
func (app *CreditApplication) BeforeCreate(tx *gorm.DB) error {
	if app.Status == "" {
		app.Status = "new"
	}
	if app.Priority == "" {
		app.Priority = "medium"
	}
	return nil
}

// AddActionToHistory adds an action to the application's history
func (app *CreditApplication) AddActionToHistory(actionType, performer, description string) {
	action := map[string]interface{}{
		"type":        actionType,
		"performer":   performer,
		"description": description,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
	}

	actions := []map[string]interface{}{}
	if app.ActionHistory != "" {
		// Try to unmarshal existing history
		if err := json.Unmarshal([]byte(app.ActionHistory), &actions); err != nil {
			// If unmarshal fails, start fresh
			actions = []map[string]interface{}{}
		}
	}

	actions = append(actions, action)

	// Marshal back to JSON
	if jsonBytes, err := json.Marshal(actions); err == nil {
		app.ActionHistory = string(jsonBytes)
	}
}
