package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"credit_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreditManagerHandler struct {
	db *gorm.DB
}

func NewCreditManagerHandler(db *gorm.DB) *CreditManagerHandler {
	return &CreditManagerHandler{db: db}
}

type debtBurdenDetails struct {
	TotalMonthlyPayments float64 `json:"total_monthly_payments"`
}

type restructureRecommendation struct {
	Decision         string   `json:"decision"`
	Summary          string   `json:"summary"`
	RecommendedPlan  string   `json:"recommendedPlan"`
	NewPayment       float64  `json:"newPayment"`
	PaymentReduction float64  `json:"paymentReduction"`
	NewTermMonths    int      `json:"newTermMonths"`
	GraceMonths      int      `json:"graceMonths"`
	FinancialState   string   `json:"financialState"`
	RiskLevel        string   `json:"riskLevel"`
	Reasons          []string `json:"reasons"`
}

type officialIncomeInfo struct {
	Salary                float64 `json:"salary"`
	Pension               float64 `json:"pension"`
	OtherOfficialPayments float64 `json:"otherOfficialPayments"`
	TotalOfficialIncome   float64 `json:"totalOfficialIncome"`
	Source                string  `json:"source"`
	CheckedAt             string  `json:"checkedAt"`
}

type restructureRequest struct {
	NewPayment    float64 `json:"newPayment"`
	NewTermMonths int     `json:"newTermMonths"`
	GraceMonths   int     `json:"graceMonths"`
}

func (h *CreditManagerHandler) GetProblemClients(c *gin.Context) {
	var apps []models.CreditApplication
	problemStatuses := []string{"approved", "signing", "contract_sent", "issued"}
	if err := h.db.
		Where("status IN ? AND current_delinquencies = ?", problemStatuses, true).
		Order("current_delinquencies DESC, max_delinquency_days DESC, dti_ratio DESC, risk_score DESC").
		Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]gin.H, 0, len(apps))
	for _, app := range apps {
		rec := buildRestructureRecommendation(app)
		items = append(items, gin.H{
			"id":                   app.ID,
			"clientName":           app.ClientName,
			"phone":                app.Phone,
			"creditType":           app.CreditType,
			"requestedAmount":      app.RequestedAmount,
			"approvedAmount":       effectiveApprovedAmount(app),
			"creditTerm":           app.CreditTerm,
			"monthlyPayment":       app.MonthlyPayment,
			"monthlyIncome":        app.MonthlyIncome,
			"totalDebt":            app.TotalDebt,
			"dtiRatio":             app.DTIRatio,
			"monthlyPayments":      monthlyPayments(app),
			"delayedPayments12m":   app.DelayedPayments12m,
			"currentDelinquencies": app.CurrentDelinquencies,
			"maxDelinquencyDays":   app.MaxDelinquencyDays,
			"riskScore":            app.RiskScore,
			"financialState":       rec.FinancialState,
			"recommendation":       rec,
		})
	}

	c.JSON(http.StatusOK, items)
}

func (h *CreditManagerHandler) GetClientAssessment(c *gin.Context) {
	app, ok := h.findApplication(c)
	if !ok {
		return
	}

	rec := buildRestructureRecommendation(app)
	c.JSON(http.StatusOK, gin.H{
		"application": gin.H{
			"id":                   app.ID,
			"clientName":           app.ClientName,
			"phone":                app.Phone,
			"email":                app.Email,
			"creditType":           app.CreditType,
			"requestedAmount":      app.RequestedAmount,
			"approvedAmount":       effectiveApprovedAmount(app),
			"creditTerm":           app.CreditTerm,
			"monthlyPayment":       app.MonthlyPayment,
			"monthlyIncome":        app.MonthlyIncome,
			"totalDebt":            app.TotalDebt,
			"dtiRatio":             app.DTIRatio,
			"monthlyPayments":      monthlyPayments(app),
			"activeCredits":        app.ActiveCredits,
			"delayedPayments12m":   app.DelayedPayments12m,
			"currentDelinquencies": app.CurrentDelinquencies,
			"maxDelinquencyDays":   app.MaxDelinquencyDays,
			"riskScore":            app.RiskScore,
			"actionHistory":        app.ActionHistory,
		},
		"recommendation": rec,
	})
}

func (h *CreditManagerHandler) RequestOfficialIncomeInfo(c *gin.Context) {
	app, ok := h.findApplication(c)
	if !ok {
		return
	}

	incomeInfo := buildOfficialIncomeInfo(app)
	username := c.GetString("username")
	if username == "" {
		username = "Кредитный менеджер"
	}

	details := fmt.Sprintf("Запрошены сведения о зарплате, пенсиях и иных официальных выплатах. Подтвержденный официальный доход: %.0f ₽.",
		incomeInfo.TotalOfficialIncome)
	app.AddActionToHistory("income_info_requested", username, details)
	app.Notes = strings.TrimSpace(app.Notes + "\n" + time.Now().Format("2006-01-02 15:04") + " " + details)

	if err := h.db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Сведения о доходах запрошены",
		"incomeInfo": incomeInfo,
	})
}

func (h *CreditManagerHandler) RestructureClient(c *gin.Context) {
	app, ok := h.findApplication(c)
	if !ok {
		return
	}

	rec := buildRestructureRecommendation(app)
	if rec.Decision == "not_recommended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Реструктуризация не рекомендована", "recommendation": rec})
		return
	}

	var req restructureRequest
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
	}
	if req.NewPayment <= 0 {
		req.NewPayment = rec.NewPayment
	}
	if req.NewTermMonths <= 0 {
		req.NewTermMonths = rec.NewTermMonths
	}
	if req.GraceMonths < 0 {
		req.GraceMonths = 0
	}
	if req.NewPayment <= 0 || req.NewTermMonths <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите новый платеж и срок кредита"})
		return
	}

	username := c.GetString("username")
	if username == "" {
		username = "Кредитный менеджер"
	}
	details := fmt.Sprintf("Предложение новых условий кредита отправлено клиенту: %s. Новый платеж: %.0f ₽, новый срок: %d мес., льготный период: %d мес.",
		rec.RecommendedPlan, req.NewPayment, req.NewTermMonths, req.GraceMonths)
	payload := map[string]interface{}{
		"application_id":     app.ID,
		"new_payment":        req.NewPayment,
		"new_term_months":    req.NewTermMonths,
		"grace_months":       req.GraceMonths,
		"recommended_plan":   rec.RecommendedPlan,
		"recommended_reason": rec.Summary,
	}
	if err := sendRestructureToClientApp(payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	app.AddActionToHistory("restructure_offer_sent", username, details)
	app.Notes = strings.TrimSpace(app.Notes + "\n" + time.Now().Format("2006-01-02 15:04") + " " + details)

	if err := h.db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Новые условия отправлены клиенту на рассмотрение", "recommendation": rec})
}

func sendRestructureToClientApp(payload map[string]interface{}) error {
	baseURL := os.Getenv("CLIENT_APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := strings.TrimRight(baseURL, "/") + "/api/restructure/receive/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Contract-Token", contractDeliveryToken())
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("клиентское приложение недоступно: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("клиентское приложение вернуло HTTP %d", resp.StatusCode)
	}
	return nil
}

func (h *CreditManagerHandler) findApplication(c *gin.Context) (models.CreditApplication, bool) {
	var app models.CreditApplication
	if err := h.db.Where("id = ?", c.Param("id")).First(&app).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return app, false
	}
	return app, true
}

func buildRestructureRecommendation(app models.CreditApplication) restructureRecommendation {
	payments := monthlyPayments(app)
	if payments == 0 && app.MonthlyIncome > 0 && app.DTIRatio > 0 {
		payments = app.MonthlyIncome * app.DTIRatio
	}

	dti := app.DTIRatio
	if dti == 0 && app.MonthlyIncome > 0 {
		dti = payments / app.MonthlyIncome
	}

	reasons := []string{}
	if app.CurrentDelinquencies {
		reasons = append(reasons, "есть текущая просрочка")
	}
	if app.DelayedPayments12m > 0 {
		reasons = append(reasons, fmt.Sprintf("просрочек за 12 месяцев: %d", app.DelayedPayments12m))
	}
	if dti >= 0.5 {
		reasons = append(reasons, fmt.Sprintf("критическая долговая нагрузка %.1f%%", dti*100))
	} else if dti >= 0.35 {
		reasons = append(reasons, fmt.Sprintf("повышенная долговая нагрузка %.1f%%", dti*100))
	}
	if app.RiskScore >= 75 {
		reasons = append(reasons, fmt.Sprintf("высокий риск %d%%", app.RiskScore))
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "критичных признаков неплатежеспособности не выявлено")
	}

	financialState := "Стабильное"
	riskLevel := "Низкий"
	decision := "monitor"
	plan := "Наблюдение без изменения условий"
	targetDti := 0.32
	graceMonths := 0
	newTerm := app.CreditTerm
	newPayment := payments

	switch {
	case app.CurrentDelinquencies || app.MaxDelinquencyDays >= 60 || dti >= 0.55:
		financialState = "Кризисное"
		riskLevel = "Высокий"
		decision = "recommended"
		plan = "Изменить условия действующего кредита: снизить ежемесячный платеж и дать льготный период"
		targetDti = 0.30
		graceMonths = 3
		newTerm = maxInt(app.CreditTerm+18, 24)
	case app.DelayedPayments12m > 0 || dti >= 0.35 || (app.RiskScore >= 75 && dti >= 0.25):
		financialState = "Напряженное"
		riskLevel = "Средний"
		decision = "recommended"
		plan = "Изменить условия действующего кредита: увеличить срок для снижения платежа"
		targetDti = 0.33
		graceMonths = 1
		newTerm = maxInt(app.CreditTerm+12, 18)
	}

	if app.MonthlyIncome > 0 && decision == "recommended" {
		newPayment = app.MonthlyIncome * targetDti
		if payments > 0 {
			newPayment = math.Min(payments*0.75, newPayment)
		}
	}
	if payments == 0 && !app.CurrentDelinquencies && app.DelayedPayments12m == 0 && dti < 0.35 {
		decision = "monitor"
		plan = "Проверить актуальную платежную информацию без изменения условий"
		newPayment = 0
		graceMonths = 0
	}
	if newPayment < 0 {
		newPayment = 0
	}
	reduction := 0.0
	if payments > 0 && newPayment < payments {
		reduction = (payments - newPayment) * 100 / payments
	}

	if app.MaxDelinquencyDays >= 90 && app.MonthlyIncome <= 0 {
		decision = "not_recommended"
		plan = "Передать на индивидуальное урегулирование"
		financialState = "Критическое"
		riskLevel = "Критический"
	}

	return restructureRecommendation{
		Decision:         decision,
		Summary:          recommendationSummary(decision),
		RecommendedPlan:  plan,
		NewPayment:       math.Round(newPayment),
		PaymentReduction: math.Round(reduction*10) / 10,
		NewTermMonths:    newTerm,
		GraceMonths:      graceMonths,
		FinancialState:   financialState,
		RiskLevel:        riskLevel,
		Reasons:          reasons,
	}
}

func recommendationSummary(decision string) string {
	switch decision {
	case "recommended":
		return "Реструктуризация целесообразна"
	case "not_recommended":
		return "Автоматическая реструктуризация не рекомендована"
	default:
		return "Достаточно мониторинга"
	}
}

func monthlyPayments(app models.CreditApplication) float64 {
	if app.DebtBurdenDetails == "" {
		return 0
	}
	var details debtBurdenDetails
	if err := json.Unmarshal([]byte(app.DebtBurdenDetails), &details); err != nil {
		return 0
	}
	return details.TotalMonthlyPayments
}

func effectiveApprovedAmount(app models.CreditApplication) float64 {
	if app.ApprovedAmount > 0 {
		return app.ApprovedAmount
	}
	return app.RequestedAmount
}

func buildOfficialIncomeInfo(app models.CreditApplication) officialIncomeInfo {
	salary := 0.0
	pension := 0.0
	other := 0.0
	employment := strings.ToLower(app.EmploymentStatus + " " + app.Position)

	if strings.Contains(employment, "пенсион") || app.Age >= 63 {
		pension = math.Round(math.Max(app.MonthlyIncome*0.75, 22000))
		if app.MonthlyIncome > pension {
			salary = math.Round((app.MonthlyIncome - pension) * 0.85)
		}
	} else {
		salary = math.Round(app.MonthlyIncome * 0.88)
	}

	if app.Age >= 55 && app.Age < 63 {
		other = 12000
	}
	if strings.Contains(strings.ToLower(app.AdditionalIncome), "да") {
		other += math.Round(app.MonthlyIncome * 0.08)
	}

	return officialIncomeInfo{
		Salary:                salary,
		Pension:               pension,
		OtherOfficialPayments: other,
		TotalOfficialIncome:   salary + pension + other,
		Source:                "ФНС/СФР/ЕГИССО (демо-запрос)",
		CheckedAt:             time.Now().Format("02.01.2006 15:04"),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
