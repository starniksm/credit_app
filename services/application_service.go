package services

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math"
	"math/rand"
	"strings"
	"time"

	"credit_app/models"
	"credit_app/utils"

	"gorm.io/gorm"
)

// calculateAge calculates age from birth date
func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}

// ApplicationService handles business logic for credit applications
type ApplicationService struct {
	db *gorm.DB
}

// NewApplicationService creates a new instance of ApplicationService
func NewApplicationService(db *gorm.DB) *ApplicationService {
	return &ApplicationService{
		db: db,
	}
}

// CreateApplication creates a new credit application
func (s *ApplicationService) CreateApplication(app *models.CreditApplication) (*models.CreditApplication, error) {
	// Set initial status and timestamps
	if app.Status == "" {
		app.Status = "new"
	}
	if app.Priority == "" {
		app.Priority = "medium"
	}
	app.CreatedAt = time.Now()
	app.UpdatedAt = time.Now()

	// Generate unique ID if not provided
	if app.ID == "" {
		app.ID = utils.GenerateCreditID()
	}

	// Set full name from client name if not provided
	if app.FullName == "" && app.ClientName != "" {
		app.FullName = app.ClientName
	}

	// Automatically enrich missing client data from a stubbed external profile service.
	s.enrichMissingClientData(app)

	// Calculate age from birth date if provided
	if app.BirthDate != nil && app.Age == 0 {
		app.Age = calculateAge(*app.BirthDate)
	}

	// Calculate initial scores and data
	s.calculateInitialScores(app)

	// Generate BKI data and AI recommendations
	s.generateBKIData(app)
	s.generateAIRecommendations(app)

	// Fetch credit history from external service
	passportData := app.PassportSeries + " " + app.PassportNumber
	creditHistory, err := utils.FetchCreditHistory(passportData)
	if err != nil {
		// Log error but continue with application creation
		fmt.Printf("Warning: Failed to fetch credit history: %v\n", err)
		creditHistory = "Credit history unavailable"
	}
	app.CreditHistory = creditHistory

	if err := s.db.Create(app).Error; err != nil {
		return nil, err
	}

	return app, nil
}

func removeActionHistoryType(app *models.CreditApplication, actionType string) bool {
	if strings.TrimSpace(app.ActionHistory) == "" {
		return false
	}
	var actions []map[string]interface{}
	if err := json.Unmarshal([]byte(app.ActionHistory), &actions); err != nil {
		return false
	}
	filtered := make([]map[string]interface{}, 0, len(actions))
	removed := false
	for _, action := range actions {
		if action["type"] == actionType {
			removed = true
			continue
		}
		filtered = append(filtered, action)
	}
	if !removed {
		return false
	}
	jsonBytes, err := json.Marshal(filtered)
	if err != nil {
		return false
	}
	app.ActionHistory = string(jsonBytes)
	return true
}

func (s *ApplicationService) enrichMissingClientData(app *models.CreditApplication) []string {
	profile := mockClientProfile(app)
	filled := make([]string, 0)

	setString := func(label string, target *string, value string) {
		if strings.TrimSpace(*target) == "" && value != "" {
			*target = value
			filled = append(filled, label)
		}
	}
	setFloat := func(label string, target *float64, value float64) {
		if *target <= 0 && value > 0 {
			*target = value
			filled = append(filled, label)
		}
	}
	setInt := func(label string, target *int, value int) {
		if *target <= 0 && value > 0 {
			*target = value
			filled = append(filled, label)
		}
	}
	setTime := func(label string, target **time.Time, value time.Time) {
		if *target == nil {
			v := value
			*target = &v
			filled = append(filled, label)
		}
	}

	setString("ФИО", &app.FullName, profile.FullName)
	setString("ФИО клиента", &app.ClientName, profile.FullName)
	setString("телефон", &app.Phone, profile.Phone)
	setString("email", &app.Email, profile.Email)
	setTime("дата рождения", &app.BirthDate, profile.BirthDate)
	setString("пол", &app.Gender, profile.Gender)
	setString("серия паспорта", &app.PassportSeries, profile.PassportSeries)
	setString("номер паспорта", &app.PassportNumber, profile.PassportNumber)
	setTime("дата выдачи паспорта", &app.PassportIssueDate, profile.PassportIssueDate)
	setString("код подразделения", &app.PassportDepartmentCode, profile.PassportDepartmentCode)
	setString("адрес регистрации", &app.RegistrationAddress, profile.RegistrationAddress)
	setString("адрес проживания", &app.ResidenceAddress, profile.ResidenceAddress)
	setString("срок проживания", &app.ResidenceDuration, profile.ResidenceDuration)
	setString("статус занятости", &app.EmploymentStatus, profile.EmploymentStatus)
	setString("тип занятости", &app.EmploymentType, profile.EmploymentType)
	setString("место работы", &app.EmployerName, profile.EmployerName)
	setString("адрес работодателя", &app.EmployerAddress, profile.EmployerAddress)
	setString("должность", &app.Position, profile.Position)
	setString("стаж на текущем месте", &app.CurrentJobExperience, profile.CurrentJobExperience)
	setString("общий стаж", &app.TotalExperience, profile.TotalExperience)
	setString("дополнительный доход", &app.AdditionalIncome, profile.AdditionalIncome)
	setFloat("ежемесячный доход", &app.MonthlyIncome, profile.MonthlyIncome)
	setFloat("ежемесячные расходы", &app.Expenses, profile.Expenses)
	setInt("стаж работы", &app.WorkExperience, profile.WorkExperience)

	if app.BirthDate != nil && app.Age == 0 {
		app.Age = calculateAge(*app.BirthDate)
		filled = append(filled, "возраст")
	}

	return filled
}

type mockProfile struct {
	FullName               string
	Phone                  string
	Email                  string
	BirthDate              time.Time
	Gender                 string
	PassportSeries         string
	PassportNumber         string
	PassportIssueDate      time.Time
	PassportDepartmentCode string
	RegistrationAddress    string
	ResidenceAddress       string
	ResidenceDuration      string
	EmploymentStatus       string
	EmploymentType         string
	EmployerName           string
	EmployerAddress        string
	Position               string
	CurrentJobExperience   string
	TotalExperience        string
	AdditionalIncome       string
	MonthlyIncome          float64
	Expenses               float64
	WorkExperience         int
}

func mockClientProfile(app *models.CreditApplication) mockProfile {
	name := firstNonEmpty(app.FullName, app.ClientName, "Иванов Иван Сергеевич")
	phone := firstNonEmpty(app.Phone, "+79991234567")
	email := firstNonEmpty(app.Email, "client@example.com")
	seed := crc32.ChecksumIEEE([]byte(name + phone + email))

	birthYear := 1975 + int(seed%24)
	birthMonth := time.Month(1 + int(seed%12))
	birthDay := 1 + int(seed%27)
	birthDate := time.Date(birthYear, birthMonth, birthDay, 0, 0, 0, 0, time.UTC)
	passportIssueDate := birthDate.AddDate(20+int(seed%18), int(seed%12), int(seed%20))

	employers := []string{"ООО «Вектор»", "АО «Технопарк»", "ООО «Север Логистик»", "АО «Городские сервисы»", "ООО «Финтрейд»"}
	positions := []string{"ведущий специалист", "менеджер проектов", "инженер", "аналитик", "руководитель группы"}
	streets := []string{"Ленинский проспект", "ул. Новаторов", "ул. Академика Королева", "Пресненская набережная", "ул. Профсоюзная"}

	passportSeries := fmt.Sprintf("%04d", 4500+seed%500)
	passportNumber := fmt.Sprintf("%06d", 100000+seed%800000)
	departmentCode := fmt.Sprintf("%03d-%03d", 100+seed%800, 100+(seed/7)%800)
	street := streets[int(seed)%len(streets)]
	house := 1 + int(seed%120)
	flat := 1 + int((seed/11)%240)
	address := fmt.Sprintf("г. Москва, %s, д. %d, кв. %d", street, house, flat)
	employer := employers[int(seed)%len(employers)]
	income := 85000 + float64(seed%90000)
	workExperience := 2 + int(seed%18)

	return mockProfile{
		FullName:               name,
		Phone:                  phone,
		Email:                  email,
		BirthDate:              birthDate,
		Gender:                 mockGender(name, seed),
		PassportSeries:         passportSeries,
		PassportNumber:         passportNumber,
		PassportIssueDate:      passportIssueDate,
		PassportDepartmentCode: departmentCode,
		RegistrationAddress:    address,
		ResidenceAddress:       address,
		ResidenceDuration:      fmt.Sprintf("%d лет", 3+seed%12),
		EmploymentStatus:       "employed",
		EmploymentType:         "full_time",
		EmployerName:           employer,
		EmployerAddress:        fmt.Sprintf("г. Москва, %s, д. %d", street, house+3),
		Position:               positions[int(seed/3)%len(positions)],
		CurrentJobExperience:   fmt.Sprintf("%d лет", 1+seed%7),
		TotalExperience:        fmt.Sprintf("%d лет", workExperience),
		AdditionalIncome:       "нет",
		MonthlyIncome:          income,
		Expenses:               income * 0.42,
		WorkExperience:         workExperience,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mockGender(name string, seed uint32) string {
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "на ") || strings.Contains(lowerName, "ова") || strings.Contains(lowerName, "ева") || strings.Contains(lowerName, "ая") {
		return "female"
	}
	if seed%2 == 0 {
		return "male"
	}
	return "female"
}

// GetApplicationByID retrieves a credit application by its ID
func (s *ApplicationService) GetApplicationByID(id string) (*models.CreditApplication, error) {
	var app models.CreditApplication
	if err := s.db.Where("id = ?", id).First(&app).Error; err != nil {
		return nil, err
	}

	enrichedFields := s.enrichMissingClientData(&app)
	historyChanged := removeActionHistoryType(&app, "client_data_enriched")
	recalculatedDebtBurden := false
	if len(enrichedFields) > 0 && app.CreditScore != 0 {
		s.recalculateDebtBurden(&app)
		if app.RecommendationReason != "recommendation_letter" {
			s.generateAIRecommendations(&app)
		}
		recalculatedDebtBurden = true
	}

	// If the application doesn't have BKI data, generate it
	if app.CreditScore == 0 {
		s.generateBKIData(&app)
		s.generateAIRecommendations(&app)
		passportData := app.PassportSeries + " " + app.PassportNumber
		creditHistory, err := utils.FetchCreditHistory(passportData)
		if err != nil {
			// Log error but continue with existing data
			fmt.Printf("Warning: Failed to fetch credit history: %v\n", err)
			creditHistory = "Credit history unavailable"
		}
		app.CreditHistory = creditHistory

		// Save the updated application with generated data
		if err := s.db.Save(&app).Error; err != nil {
			return nil, err
		}
	} else if historyChanged || recalculatedDebtBurden || (app.MonthlyIncome > 0 && app.DTIRatio == 0 && strings.TrimSpace(app.DebtBurdenDetails) != "") {
		if !recalculatedDebtBurden {
			s.recalculateDebtBurden(&app)
			if app.RecommendationReason != "recommendation_letter" {
				s.generateAIRecommendations(&app)
			}
		}
		if err := s.db.Save(&app).Error; err != nil {
			return nil, err
		}
	}

	return &app, nil
}

// GetApplications retrieves all credit applications with optional filtering
func (s *ApplicationService) GetApplications(filter *models.ApplicationFilter) ([]*models.CreditApplication, error) {
	var applications []*models.CreditApplication
	query := s.db.Model(&models.CreditApplication{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CreditType != "" {
		query = query.Where("credit_type = ?", filter.CreditType)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("full_name LIKE ? OR client_name LIKE ? OR phone LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if filter.SortBy != "" {
		orderClause := filter.SortBy
		if strings.ToLower(filter.SortOrder) == "desc" {
			orderClause += " DESC"
		}
		query = query.Order(orderClause)
	} else {
		query = query.Order("created_at DESC") // Default ordering
	}

	if filter.Limit > 0 {
		query = query.Limit(int(filter.Limit))
		if filter.Offset > 0 {
			query = query.Offset(int(filter.Offset))
		}
	}

	if err := query.Find(&applications).Error; err != nil {
		return nil, err
	}

	return applications, nil
}

// UpdateApplication updates an existing credit application
func (s *ApplicationService) UpdateApplication(id string, app *models.CreditApplication) (*models.CreditApplication, error) {
	var existingApp models.CreditApplication
	if err := s.db.Where("id = ?", id).First(&existingApp).Error; err != nil {
		return nil, err
	}

	// Use a transaction to ensure data consistency
	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update only the fields that are provided in the request
	if app.ClientName != "" {
		existingApp.ClientName = app.ClientName
	}
	if app.FullName != "" {
		existingApp.FullName = app.FullName
	}
	if app.Phone != "" {
		existingApp.Phone = app.Phone
	}
	if app.Email != "" {
		existingApp.Email = app.Email
	}
	if app.PassportSeries != "" {
		existingApp.PassportSeries = app.PassportSeries
	}
	if app.PassportNumber != "" {
		existingApp.PassportNumber = app.PassportNumber
	}
	if app.RequestedAmount > 0 {
		existingApp.RequestedAmount = app.RequestedAmount
	}
	if app.ApprovedAmount > 0 {
		existingApp.ApprovedAmount = app.ApprovedAmount
	}
	if app.MonthlyIncome > 0 {
		existingApp.MonthlyIncome = app.MonthlyIncome
	}
	if app.Expenses > 0 {
		existingApp.Expenses = app.Expenses
	}
	if app.CreditPurpose != "" {
		existingApp.CreditPurpose = app.CreditPurpose
	}
	if app.CreditType != "" {
		existingApp.CreditType = app.CreditType
	}
	if app.RepaymentTerm > 0 {
		existingApp.RepaymentTerm = app.RepaymentTerm
	}
	if app.EmploymentType != "" {
		existingApp.EmploymentType = app.EmploymentType
	}
	if app.WorkExperience > 0 {
		existingApp.WorkExperience = app.WorkExperience
	}
	if app.Position != "" {
		existingApp.Position = app.Position
	}
	if app.EmployerName != "" {
		existingApp.EmployerName = app.EmployerName
	}
	if app.EmployerAddress != "" {
		existingApp.EmployerAddress = app.EmployerAddress
	}
	if app.Status != "" {
		existingApp.Status = app.Status
	}
	if app.Priority != "" {
		existingApp.Priority = app.Priority
	}
	if app.Notes != "" {
		existingApp.Notes = app.Notes
	}
	if app.Documents != nil {
		existingApp.Documents = app.Documents
	}
	if app.DecisionReason != "" {
		existingApp.DecisionReason = app.DecisionReason
	}
	if app.ReviewerID != "" {
		existingApp.ReviewerID = app.ReviewerID
	}
	if app.AnalystName != "" {
		existingApp.AnalystName = app.AnalystName
	}
	if app.ReviewStartedAt != nil {
		existingApp.ReviewStartedAt = app.ReviewStartedAt
	}
	if app.ReviewCompletedAt != nil {
		existingApp.ReviewCompletedAt = app.ReviewCompletedAt
	}
	if app.LastStatusChangeAt != nil {
		existingApp.LastStatusChangeAt = app.LastStatusChangeAt
	}

	// Update BKI data
	existingApp.CreditScore = app.CreditScore
	existingApp.TotalCredits = app.TotalCredits
	existingApp.ActiveCredits = app.ActiveCredits
	existingApp.ClosedCredits = app.ClosedCredits
	existingApp.DelayedPayments12m = app.DelayedPayments12m
	existingApp.CreditHistoryScore = app.CreditHistoryScore
	existingApp.SolvabilityScore = app.SolvabilityScore
	existingApp.TotalDebt = app.TotalDebt
	existingApp.DTIRatio = app.DTIRatio
	existingApp.DebtBurdenRatio = app.DebtBurdenRatio
	existingApp.RiskScore = app.RiskScore
	existingApp.AIRecommendation = app.AIRecommendation
	existingApp.AIScore = app.AIScore
	existingApp.AIComment = app.AIComment
	existingApp.CreditHistory = app.CreditHistory
	existingApp.ActiveCreditsList = app.ActiveCreditsList
	existingApp.DelinquencyHistory = app.DelinquencyHistory
	existingApp.DebtBurdenDetails = app.DebtBurdenDetails
	existingApp.FactorsAnalysis = app.FactorsAnalysis
	existingApp.ActionHistory = app.ActionHistory
	removeActionHistoryType(&existingApp, "client_data_enriched")
	existingApp.DecisionReason = app.DecisionReason

	if err := tx.Save(&existingApp).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return &existingApp, tx.Commit().Error
}

// DeleteApplication deletes a credit application by its ID
func (s *ApplicationService) DeleteApplication(id string) error {
	result := s.db.Where("id = ?", id).Delete(&models.CreditApplication{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// calculateInitialScores calculates initial scoring for the application
func (s *ApplicationService) calculateInitialScores(app *models.CreditApplication) {
	// Calculate debt-to-income ratio
	debtToIncome := 0.0
	if app.MonthlyIncome > 0 {
		estimatedMonthlyPayment := float64(app.RequestedAmount) / float64(app.RepaymentTerm)
		debtToIncome = estimatedMonthlyPayment / app.MonthlyIncome
	}

	// Set initial risk score based on basic factors
	baseScore := 500
	if debtToIncome < 0.3 {
		baseScore += 50
	} else if debtToIncome > 0.5 {
		baseScore -= 100
	}

	// Adjust based on employment type
	switch app.EmploymentType {
	case "Штатное":
		baseScore += 50
	case "Фриланс":
		baseScore -= 30
	case "Самозанятый":
		baseScore -= 20
	}

	// Adjust based on work experience
	if app.WorkExperience > 5 {
		baseScore += 30
	} else if app.WorkExperience < 2 {
		baseScore -= 20
	}

	// Adjust based on requested amount
	if float64(app.RequestedAmount) > app.MonthlyIncome*10 {
		baseScore -= 50
	}

	// Ensure score is within valid range
	if baseScore < 300 {
		baseScore = 300
	} else if baseScore > 850 {
		baseScore = 850
	}

	app.InitialScore = baseScore
}

// generateBKIData generates realistic BKI data for the application
func (s *ApplicationService) generateBKIData(app *models.CreditApplication) {
	providedActiveCredits := make([]map[string]interface{}, 0)
	if strings.TrimSpace(app.ActiveCreditsList) != "" {
		if err := json.Unmarshal([]byte(app.ActiveCreditsList), &providedActiveCredits); err != nil {
			providedActiveCredits = make([]map[string]interface{}, 0)
		}
	}
	providedActiveCredits = filterProvidedActiveCredits(providedActiveCredits)
	providedActiveCreditsCount := len(providedActiveCredits)
	providedTotalDebt := 0.0
	providedMonthlyPayments := 0.0
	for _, credit := range providedActiveCredits {
		if debt, ok := credit["debt"].(float64); ok {
			providedTotalDebt += debt
		}
		if monthlyPayment, ok := credit["monthly_payment"].(float64); ok {
			providedMonthlyPayments += monthlyPayment
		}
	}
	if providedActiveCreditsCount == 0 && strings.TrimSpace(app.DebtBurdenDetails) != "" {
		// Fallback for legacy payloads that had debt details but no active credits list.
		var providedDebtDetails map[string]interface{}
		if err := json.Unmarshal([]byte(app.DebtBurdenDetails), &providedDebtDetails); err == nil {
			if existingPayment, ok := providedDebtDetails["existing_monthly_payments"].(float64); ok {
				providedMonthlyPayments = existingPayment
			} else if monthlyPayment, ok := providedDebtDetails["total_monthly_payments"].(float64); ok {
				providedMonthlyPayments = monthlyPayment
			}
		}
	}

	// Generate credit score (300-850) based on initial score with some randomness
	creditScore := app.InitialScore
	creditScore += rand.Intn(100) - 50 // Add random variation
	if creditScore < 300 {
		creditScore = 300
	} else if creditScore > 850 {
		creditScore = 850
	}

	// Generate total credits count (0-15) and keep credits provided by the client app.
	generatedTotalCredits := rand.Intn(16)
	generatedActiveCredits := 0
	closedCredits := 0

	// Distribute between active and closed credits
	if generatedTotalCredits > 0 {
		closedCredits = rand.Intn(generatedTotalCredits)
		generatedActiveCredits = generatedTotalCredits - closedCredits
	}
	activeCredits := generatedActiveCredits + providedActiveCreditsCount
	totalCredits := generatedTotalCredits + providedActiveCreditsCount

	// Generate delayed payments (0-20)
	delayedPayments12m := rand.Intn(21)

	// Generate credit history score (300-850)
	creditHistoryScore := creditScore - rand.Intn(50)

	// Generate solvability score (0-100)
	solvabilityScore := 50 + rand.Intn(51) - 25 // Base 50 with variation
	if solvabilityScore < 0 {
		solvabilityScore = 0
	} else if solvabilityScore > 100 {
		solvabilityScore = 100
	}

	// Generate active credits list and append credits supplied by the client app.
	activeCreditsList := make([]map[string]interface{}, generatedActiveCredits)
	generatedTotalDebt := 0.0
	generatedMonthlyPayments := 0.0
	for i := 0; i < generatedActiveCredits; i++ {
		bankNames := []string{"Сбербанк", "ВТБ", "Газпромбанк", "Альфа-Банк", "Тинькофф", "Росбанк", "Совкомбанк"}
		creditTypes := []string{"Потребительский", "Ипотечный", "Автокредит", "Кредитная карта"}

		daysOverdue := 0
		status := "ok"
		if rand.Float64() < 0.3 { // 30% chance of having some issues
			daysOverdue = rand.Intn(180)
			if daysOverdue > 90 {
				status = "critical"
			} else if daysOverdue > 0 {
				status = "delayed"
			}
		}

		// Remove unused variable
		// paymentStatuses := []string{"ok", "delayed", "critical"}

		debt := float64(rand.Intn(500000) + 50000)         // 50k to 550k
		monthlyPayment := float64(rand.Intn(50000) + 5000) // 5k to 55k
		generatedTotalDebt += debt
		generatedMonthlyPayments += monthlyPayment

		activeCreditsList[i] = map[string]interface{}{
			"bank":            bankNames[rand.Intn(len(bankNames))],
			"type":            creditTypes[rand.Intn(len(creditTypes))],
			"debt":            debt,
			"monthly_payment": monthlyPayment,
			"payment_status":  status,
			"days_overdue":    daysOverdue,
		}
	}
	activeCreditsList = append(activeCreditsList, providedActiveCredits...)
	existingMonthlyPayments := generatedMonthlyPayments + providedMonthlyPayments
	newLoanMonthlyPayment := calculateLoanMonthlyPayment(app.RequestedAmount, effectiveTermMonths(app), 14)
	if app.MonthlyPayment > 0 {
		newLoanMonthlyPayment = app.MonthlyPayment
	} else {
		app.MonthlyPayment = newLoanMonthlyPayment
	}
	totalDebt := generatedTotalDebt + providedTotalDebt
	totalMonthlyPayments := existingMonthlyPayments + newLoanMonthlyPayment

	// DTI includes existing obligations plus the requested loan payment.
	debtBurdenRatio := 0.0
	if app.MonthlyIncome > 0 {
		debtBurdenRatio = totalMonthlyPayments / app.MonthlyIncome
	}

	// Generate risk score based on multiple factors
	riskScore := s.calculateRiskScore(app, creditScore, debtBurdenRatio, delayedPayments12m)

	// Generate delinquency history
	delinqCount := delayedPayments12m + rand.Intn(10)
	delinqHistory := make([]map[string]interface{}, delinqCount)
	for i := 0; i < delinqCount; i++ {
		delinqHistory[i] = map[string]interface{}{
			"type":     "Просрочка платежа",
			"amount":   float64(rand.Intn(100000) + 10000),
			"days":     rand.Intn(180),
			"date":     time.Now().AddDate(0, -rand.Intn(24), -rand.Intn(30)).Format("2006-01-02"),
			"resolved": rand.Float64() > 0.4, // 60% chance it's resolved
		}
	}

	// Generate debt burden details
	debtBurdenDetails := map[string]interface{}{
		"existing_monthly_payments":  existingMonthlyPayments,
		"new_loan_monthly_payment":   newLoanMonthlyPayment,
		"total_monthly_payments":     totalMonthlyPayments,
		"monthly_income":             app.MonthlyIncome,
		"income_percentage":          debtBurdenRatio * 100,
		"recommended_max":            app.MonthlyIncome * 0.4,
		"free_income_after_payments": app.MonthlyIncome - totalMonthlyPayments,
	}

	// Convert data structures to JSON strings for storage
	activeCreditsListJSON, _ := json.Marshal(activeCreditsList)
	delinqHistoryJSON, _ := json.Marshal(delinqHistory)
	debtBurdenDetailsJSON, _ := json.Marshal(debtBurdenDetails)

	// Assign generated values to the application
	app.CreditScore = creditScore
	app.TotalCredits = totalCredits
	app.ActiveCredits = activeCredits
	app.ClosedCredits = closedCredits
	app.TotalDebt = totalDebt
	app.DelayedPayments12m = delayedPayments12m
	app.CreditHistoryScore = creditHistoryScore
	app.SolvabilityScore = solvabilityScore
	app.DTIRatio = debtBurdenRatio
	app.DebtBurdenRatio = debtBurdenRatio
	app.RiskScore = riskScore
	app.ActiveCreditsList = string(activeCreditsListJSON)
	app.DelinquencyHistory = string(delinqHistoryJSON)
	app.DebtBurdenDetails = string(debtBurdenDetailsJSON)
}

func effectiveTermMonths(app *models.CreditApplication) int {
	if app.RepaymentTerm > 0 {
		return app.RepaymentTerm
	}
	if app.CreditTerm > 0 {
		return app.CreditTerm
	}
	return 12
}

func calculateLoanMonthlyPayment(amount float64, termMonths int, annualRate float64) float64 {
	if amount <= 0 || termMonths <= 0 {
		return 0
	}
	monthlyRate := annualRate / 100 / 12
	if monthlyRate <= 0 {
		return amount / float64(termMonths)
	}
	pow := math.Pow(1+monthlyRate, float64(termMonths))
	return amount * monthlyRate * pow / (pow - 1)
}

func parseDebtBurdenDetails(detailsJSON string) map[string]float64 {
	result := map[string]float64{}
	if strings.TrimSpace(detailsJSON) == "" {
		return result
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(detailsJSON), &raw); err != nil {
		return result
	}
	for key, value := range raw {
		switch v := value.(type) {
		case float64:
			result[key] = v
		case int:
			result[key] = float64(v)
		}
	}
	return result
}

func filterProvidedActiveCredits(credits []map[string]interface{}) []map[string]interface{} {
	provided := make([]map[string]interface{}, 0, len(credits))
	for _, credit := range credits {
		bank, _ := credit["bank"].(string)
		if bank == "ТБанк" || credit["initial_amount"] != nil || credit["next_payment_date"] != nil || credit["interest_rate"] != nil {
			provided = append(provided, credit)
		}
	}
	return provided
}

func (s *ApplicationService) recalculateDebtBurden(app *models.CreditApplication) {
	details := parseDebtBurdenDetails(app.DebtBurdenDetails)
	existingMonthlyPayments := details["existing_monthly_payments"]
	detailsNewLoanPayment := details["new_loan_monthly_payment"]
	newLoanMonthlyPayment := detailsNewLoanPayment
	if app.MonthlyPayment > 0 {
		newLoanMonthlyPayment = app.MonthlyPayment
	} else {
		newLoanMonthlyPayment = calculateLoanMonthlyPayment(app.RequestedAmount, effectiveTermMonths(app), 14)
		app.MonthlyPayment = newLoanMonthlyPayment
	}
	if existingMonthlyPayments == 0 {
		if totalMonthlyPayments := details["total_monthly_payments"]; totalMonthlyPayments > 0 {
			if detailsNewLoanPayment > 0 {
				existingMonthlyPayments = math.Max(totalMonthlyPayments-detailsNewLoanPayment, 0)
			} else {
				// Legacy payloads used total_monthly_payments for existing credits only.
				existingMonthlyPayments = totalMonthlyPayments
			}
		}
	}
	if existingMonthlyPayments == 0 && strings.TrimSpace(app.ActiveCreditsList) != "" {
		var activeCredits []map[string]interface{}
		if err := json.Unmarshal([]byte(app.ActiveCreditsList), &activeCredits); err == nil {
			for _, credit := range activeCredits {
				if monthlyPayment, ok := credit["monthly_payment"].(float64); ok {
					existingMonthlyPayments += monthlyPayment
				}
			}
		}
	}

	totalMonthlyPayments := existingMonthlyPayments + newLoanMonthlyPayment
	debtBurdenRatio := 0.0
	if app.MonthlyIncome > 0 {
		debtBurdenRatio = totalMonthlyPayments / app.MonthlyIncome
	}
	app.DTIRatio = debtBurdenRatio
	app.DebtBurdenRatio = debtBurdenRatio

	updatedDetails := map[string]interface{}{
		"existing_monthly_payments":  existingMonthlyPayments,
		"new_loan_monthly_payment":   newLoanMonthlyPayment,
		"total_monthly_payments":     totalMonthlyPayments,
		"monthly_income":             app.MonthlyIncome,
		"income_percentage":          debtBurdenRatio * 100,
		"recommended_max":            app.MonthlyIncome * 0.4,
		"free_income_after_payments": app.MonthlyIncome - totalMonthlyPayments,
	}
	if detailsJSON, err := json.Marshal(updatedDetails); err == nil {
		app.DebtBurdenDetails = string(detailsJSON)
	}
}

// calculateRiskScore calculates an overall risk score based on multiple factors
func (s *ApplicationService) calculateRiskScore(app *models.CreditApplication, creditScore int, debtBurdenRatio float64, delayedPayments int) int {
	// Base score from credit score (normalized to 0-100 scale)
	baseRisk := 100 - (creditScore-300)*100/550

	// Factor in debt burden (higher debt burden = higher risk)
	debtFactor := debtBurdenRatio * 50 // Scale debt burden impact

	// Factor in delayed payments (more delays = higher risk)
	delayFactor := float64(delayedPayments) * 2.5 // Each delay adds to risk

	// Factor in income vs requested amount
	incomeFactor := 0.0
	termMonths := effectiveTermMonths(app)
	if app.MonthlyIncome > 0 && termMonths > 0 {
		// Higher requested amount relative to income over the loan term = higher risk.
		amountToIncomeRatio := float64(app.RequestedAmount) / (app.MonthlyIncome * float64(termMonths))
		incomeFactor = amountToIncomeRatio * 30
	}

	// Combine factors with different weights
	finalRisk := float64(baseRisk)*0.4 + debtFactor*0.3 + delayFactor*0.2 + incomeFactor*0.1

	// Ensure score is within bounds
	if finalRisk < 0 {
		finalRisk = 0
	} else if finalRisk > 100 {
		finalRisk = 100
	}

	return int(finalRisk)
}

// generateAIRecommendations generates AI-based recommendations for the application
func (s *ApplicationService) generateAIRecommendations(app *models.CreditApplication) {

	// Generate recommendation based on risk score
	recommendation := "approve"
	aiScore := 80 // Start with a base score

	// Adjust score based on various factors
	if app.RiskScore > 70 {
		recommendation = "reject"
		aiScore = 20
	} else if app.RiskScore > 50 {
		recommendation = "request_revision"
		aiScore = 40
	} else if app.RiskScore > 30 {
		recommendation = "approve_with_conditions"
		aiScore = 60
	}

	// Further adjust based on credit score
	if app.CreditScore < 500 {
		recommendation = "reject"
		aiScore = 15
	} else if app.CreditScore > 700 {
		if recommendation != "reject" {
			aiScore += 15
			if aiScore > 95 {
				aiScore = 95
			}
		}
	}

	// Generate comment based on profile
	commentParts := []string{}
	if app.CreditScore < 500 {
		commentParts = append(commentParts, "Низкий кредитный рейтинг")
	}
	if app.RiskScore > 60 {
		commentParts = append(commentParts, "Высокий уровень риска")
	}
	if app.DebtBurdenRatio > 0.5 {
		commentParts = append(commentParts, "Высокая долговая нагрузка")
	}
	if app.DelayedPayments12m > 3 {
		commentParts = append(commentParts, "Частые просрочки по платежам")
	}
	if len(commentParts) == 0 {
		commentParts = append(commentParts, "Хороший финансовый профиль")
	}

	// Set the AI recommendation
	app.AIRecommendation = recommendation
	app.AIScore = aiScore
	app.AIComment = strings.Join(commentParts, "; ")

	// Generate factors analysis
	factors := []map[string]interface{}{}
	debtDetails := parseDebtBurdenDetails(app.DebtBurdenDetails)
	existingPayments := debtDetails["existing_monthly_payments"]
	newLoanPayment := debtDetails["new_loan_monthly_payment"]
	totalPayments := debtDetails["total_monthly_payments"]
	freeIncome := debtDetails["free_income_after_payments"]
	if totalPayments == 0 && app.MonthlyIncome > 0 && app.DebtBurdenRatio > 0 {
		totalPayments = app.MonthlyIncome * app.DebtBurdenRatio
	}
	if newLoanPayment == 0 && app.MonthlyPayment > 0 {
		newLoanPayment = app.MonthlyPayment
	}
	if freeIncome == 0 && app.MonthlyIncome > 0 {
		freeIncome = app.MonthlyIncome - totalPayments
	}

	// Positive factors
	if app.CreditScore > 700 {
		factors = append(factors, map[string]interface{}{
			"type":  "positive",
			"text":  fmt.Sprintf("Высокий кредитный рейтинг (%d)", app.CreditScore),
			"value": app.CreditScore,
		})
	}
	if app.WorkExperience > 5 {
		factors = append(factors, map[string]interface{}{
			"type":  "positive",
			"text":  fmt.Sprintf("Длительный трудовой стаж (%d лет)", app.WorkExperience),
			"value": app.WorkExperience,
		})
	}
	if app.MonthlyIncome > 100000 {
		factors = append(factors, map[string]interface{}{
			"type":  "positive",
			"text":  fmt.Sprintf("Подтвержден высокий доход: %.0f ₽", app.MonthlyIncome),
			"value": app.MonthlyIncome,
		})
	}
	if app.DebtBurdenRatio > 0 && app.DebtBurdenRatio <= 0.35 {
		factors = append(factors, map[string]interface{}{
			"type":  "positive",
			"text":  fmt.Sprintf("Комфортный DTI после выдачи: %.1f%%", app.DebtBurdenRatio*100),
			"value": app.DebtBurdenRatio * 100,
		})
	}
	if freeIncome > app.MonthlyIncome*0.45 {
		factors = append(factors, map[string]interface{}{
			"type":  "positive",
			"text":  fmt.Sprintf("Достаточный остаток дохода после платежей: %.0f ₽", freeIncome),
			"value": freeIncome,
		})
	}

	// Neutral/informational factors
	factors = append(factors, map[string]interface{}{
		"type":  "neutral",
		"text":  fmt.Sprintf("Ежемесячный доход клиента: %.0f ₽", app.MonthlyIncome),
		"value": app.MonthlyIncome,
	})
	factors = append(factors, map[string]interface{}{
		"type":  "neutral",
		"text":  fmt.Sprintf("Платежи по действующим кредитам: %.0f ₽", existingPayments),
		"value": existingPayments,
	})
	factors = append(factors, map[string]interface{}{
		"type":  "neutral",
		"text":  fmt.Sprintf("Платеж по новой заявке: %.0f ₽", newLoanPayment),
		"value": newLoanPayment,
	})
	factors = append(factors, map[string]interface{}{
		"type":  "neutral",
		"text":  fmt.Sprintf("Итоговая долговая нагрузка: %.1f%%", app.DebtBurdenRatio*100),
		"value": app.DebtBurdenRatio * 100,
	})

	// Negative factors
	if app.CreditScore < 500 {
		factors = append(factors, map[string]interface{}{
			"type":  "negative",
			"text":  fmt.Sprintf("Низкий кредитный рейтинг (%d)", app.CreditScore),
			"value": app.CreditScore,
		})
	}
	if app.DebtBurdenRatio > 0.5 {
		factors = append(factors, map[string]interface{}{
			"type":  "negative",
			"text":  fmt.Sprintf("Высокий DTI после выдачи: %.1f%%", app.DebtBurdenRatio*100),
			"value": app.DebtBurdenRatio * 100,
		})
	} else if app.DebtBurdenRatio > 0.4 {
		factors = append(factors, map[string]interface{}{
			"type":  "warning",
			"text":  fmt.Sprintf("Повышенный DTI после выдачи: %.1f%%", app.DebtBurdenRatio*100),
			"value": app.DebtBurdenRatio * 100,
		})
	}
	if freeIncome < app.MonthlyIncome*0.25 {
		factors = append(factors, map[string]interface{}{
			"type":  "negative",
			"text":  fmt.Sprintf("Низкий остаток дохода после платежей: %.0f ₽", freeIncome),
			"value": freeIncome,
		})
	}
	if app.DelayedPayments12m > 3 {
		factors = append(factors, map[string]interface{}{
			"type":  "negative",
			"text":  fmt.Sprintf("Просрочки платежей за 12 месяцев (%d)", app.DelayedPayments12m),
			"value": app.DelayedPayments12m,
		})
	}

	factorsJSON, err := json.Marshal(factors)
	if err != nil {
		fmt.Printf("Error marshaling factors analysis: %v\n", err)
		app.FactorsAnalysis = "[]"
	} else {
		app.FactorsAnalysis = string(factorsJSON)
	}
}

// calculateFinalScore calculates a final score incorporating credit history data
func (s *ApplicationService) calculateFinalScore(app *models.CreditApplication) (float64, error) {
	// Start with the AI score as base
	baseScore := float64(app.AIScore)

	// Get credit history information
	passportData := app.PassportSeries + " " + app.PassportNumber
	creditHistory, err := utils.FetchCreditHistory(passportData)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch credit history: %w", err)
	}

	// Apply adjustments based on credit history
	historyFactor := 1.0
	if strings.Contains(strings.ToLower(creditHistory), "bad") || strings.Contains(strings.ToLower(creditHistory), "defaulted") {
		historyFactor = 0.7
	} else if strings.Contains(strings.ToLower(creditHistory), "good") {
		historyFactor = 1.2
	} else if strings.Contains(strings.ToLower(creditHistory), "fair") {
		historyFactor = 0.9
	}

	finalScore := baseScore * historyFactor

	// Ensure score stays within bounds
	if finalScore < 0 {
		finalScore = 0
	} else if finalScore > 100 {
		finalScore = 100
	}

	return finalScore, nil
}

// RequestBKIData fetches and updates the BKI data for an application
func (s *ApplicationService) RequestBKIData(id string) (*models.CreditApplication, error) {
	// Get the existing application
	var existingApp models.CreditApplication
	if err := s.db.Where("id = ?", id).First(&existingApp).Error; err != nil {
		return nil, err
	}
	s.enrichMissingClientData(&existingApp)
	removeActionHistoryType(&existingApp, "client_data_enriched")

	// Simulate delay to mimic external API call
	time.Sleep(7 * time.Second) // 7 second delay to simulate processing time (between 5-10 seconds)

	// Generate new BKI data for the application
	s.generateBKIData(&existingApp)
	s.generateAIRecommendations(&existingApp)

	// Fetch updated credit history from external service
	passportData := existingApp.PassportSeries + " " + existingApp.PassportNumber
	creditHistory, err := utils.FetchCreditHistory(passportData)
	if err != nil {
		// Log error but continue with existing data
		fmt.Printf("Warning: Failed to fetch credit history: %v\n", err)
		creditHistory = "Credit history unavailable"
	}
	existingApp.CreditHistory = creditHistory

	existingApp.CreditHistory = creditHistory

	// Add BKI request events to action history
	currentTime := time.Now()
	var actionHistory []map[string]interface{}

	// Try to parse existing action history
	if existingApp.ActionHistory != "" {
		if err := json.Unmarshal([]byte(existingApp.ActionHistory), &actionHistory); err != nil {
			fmt.Printf("Error parsing action history: %v\n", err)
			actionHistory = []map[string]interface{}{}
		}
	}

	// Add new events for BKI request
	bkiRequestEvent := map[string]interface{}{
		"action":  "Отправлен запрос в БКИ",
		"author":  "Система",
		"date":    currentTime.Format(time.RFC3339),
		"comment": "Автоматический запрос кредитной истории",
		"type":    "auto",
	}

	bkiReceivedEvent := map[string]interface{}{
		"action":  "Получены данные БКИ",
		"author":  "Система",
		"date":    currentTime.Format(time.RFC3339),
		"comment": "Кредитная история загружена",
		"type":    "auto",
	}

	actionHistory = append(actionHistory, bkiRequestEvent, bkiReceivedEvent)

	// Convert back to JSON string
	historyBytes, err := json.Marshal(actionHistory)
	if err != nil {
		fmt.Printf("Error marshaling action history: %v\n", err)
		// Continue anyway, don't fail the whole operation
	} else {
		existingApp.ActionHistory = string(historyBytes)
	}

	// Save the updated application with new BKI data
	if err := s.db.Save(&existingApp).Error; err != nil {
		return nil, err
	}

	return &existingApp, nil
}

// GetDB returns the database connection
func (s *ApplicationService) GetDB() *gorm.DB {
	return s.db
}
