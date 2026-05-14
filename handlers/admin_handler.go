package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"credit_app/models"
	"credit_app/services"
	"credit_app/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	service *services.ApplicationService
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		service: services.NewApplicationService(db),
	}
}

// GetDashboardStats returns statistics for the admin dashboard
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	var stats struct {
		Total        int64  `json:"total"`
		Approved     int64  `json:"approved"`
		Rejected     int64  `json:"rejected"`
		New          int64  `json:"new"`
		InReview     int64  `json:"in_review"`
		ManualReview int64  `json:"manual_review"`
		Username     string `json:"username"`
	}

	db := h.service.GetDB()

	// Get username from context
	if username := c.GetString("username"); username != "" {
		stats.Username = username
	}

	// Count total applications
	db.Model(&models.CreditApplication{}).Count(&stats.Total)

	// Count by status
	db.Model(&models.CreditApplication{}).Where("status = ?", "approved").Count(&stats.Approved)
	db.Model(&models.CreditApplication{}).Where("status = ?", "rejected").Count(&stats.Rejected)
	db.Model(&models.CreditApplication{}).Where("status = ?", "new").Count(&stats.New)
	db.Model(&models.CreditApplication{}).Where("status = ?", "in_review").Count(&stats.InReview)
	db.Model(&models.CreditApplication{}).Where("status = ?", "manual_review").Count(&stats.ManualReview)

	c.JSON(http.StatusOK, stats)
}

// StartReview marks an application as being reviewed by an admin
func (h *AdminHandler) StartReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	reviewerID := "admin"
	if userID := c.GetUint("user_id"); userID != 0 {
		reviewerID = fmt.Sprintf("%d", userID)
	}

	// Get username from context
	username := c.GetString("username")
	if username == "" {
		username = "Администратор"
	}
	analystName := username
	if userID := c.GetUint("user_id"); userID != 0 {
		var user models.User
		if err := h.service.GetDB().First(&user, userID).Error; err == nil {
			if user.FullName != "" {
				analystName = user.FullName
			} else if user.Username != "" {
				analystName = user.Username
			}
		}
	}

	// Get the existing application
	application, err := h.service.GetApplicationByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Only allow starting review for new applications
	if application.Status != "new" && application.Status != "in_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only start review for new or in-review applications"})
		return
	}

	now := time.Now()
	application.Status = "in_review"
	application.ReviewStartedAt = &now
	application.ReviewerID = reviewerID
	application.AnalystName = analystName

	// Add action to history
	application.AddActionToHistory("started_review", analystName, "Начало проверки заявки")

	result, err := h.service.UpdateApplication(id, application)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateApplicationStatus allows admins to update the status of an application
func (h *AdminHandler) UpdateApplicationStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=approved rejected pending new in_review manual_review contract_sent issued"`
		Notes  string `json:"notes,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get username from context
	username := c.GetString("username")
	if username == "" {
		username = "Администратор"
	}

	// Get the existing application
	application, err := h.service.GetApplicationByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update the status
	application.Status = req.Status
	if req.Status == "approved" {
		if application.ApprovedAmount <= 0 || application.ApprovedAmount > application.RequestedAmount {
			application.ApprovedAmount = application.RequestedAmount
		}
	}
	if req.Status == "approved" || req.Status == "rejected" {
		now := time.Now()
		if application.ReviewStartedAt == nil {
			startedAt := application.CreatedAt
			if startedAt.IsZero() {
				startedAt = now
			}
			application.ReviewStartedAt = &startedAt
		}
		application.ReviewCompletedAt = &now
	}

	// Add action to history
	actionDetails := "Изменение статуса на: " + req.Status
	if req.Notes != "" {
		actionDetails += ". Примечание: " + req.Notes
	}
	application.AddActionToHistory("status_changed", username, actionDetails)

	// Save the updated application
	result, err := h.service.UpdateApplication(id, application)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExportApplications exports applications based on the specified format
func (h *AdminHandler) ExportApplications(c *gin.Context) {
	exportType := c.Query("type") // csv, pdf, or json
	filename := c.Query("filename")

	if filename == "" {
		filename = "credit_applications_export"
	}

	db := h.service.GetDB()

	switch exportType {
	case "csv":
		filename += ".csv"
		err := utils.ExportApplicationsToCSV(db, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "pdf":
		filename += ".pdf"
		err := utils.ExportApplicationsToPDF(db, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "json":
		filename += ".json"
		err := utils.ExportApplicationsToJSON(db, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export type. Use csv, pdf, or json"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Export completed successfully", "filename": filename})
}

// GetApplicationsForReview returns applications that need review (new or in_review)
func (h *AdminHandler) GetApplicationsForReview(c *gin.Context) {
	var filter models.ApplicationFilter
	filter.Status = "" // Get all applications, we'll filter in the query

	applications, err := h.service.GetApplications(&filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, applications)
}

// DecisionRequest represents a request to make a decision on an application
type DecisionRequest struct {
	Decision string `json:"decision" binding:"required,oneof=approved rejected"`
	Reason   string `json:"reason,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// MakeDecision handles approving or rejecting a credit application
func (h *AdminHandler) MakeDecision(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var req DecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get username from context
	username := c.GetString("username")
	if username == "" {
		username = "Администратор"
	}

	// Get the existing application
	application, err := h.service.GetApplicationByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Only allow decisions for applications in review
	if application.Status != "in_review" && application.Status != "new" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only make decisions for applications in review"})
		return
	}

	// Validate rejection reason if decision is reject
	if req.Decision == "rejected" && req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required for rejection"})
		return
	}

	// Update the application status
	application.Status = req.Decision

	// If approved, set approved amount to requested amount
	if req.Decision == "approved" {
		application.ApprovedAmount = application.RequestedAmount
	}
	now := time.Now()
	if application.ReviewStartedAt == nil {
		startedAt := application.CreatedAt
		if startedAt.IsZero() {
			startedAt = now
		}
		application.ReviewStartedAt = &startedAt
	}
	application.ReviewCompletedAt = &now

	// Set decision reason
	application.DecisionReason = req.Reason

	// Add action to history with Russian text
	decisionText := "Одобрено"
	if req.Decision == "rejected" {
		decisionText = "Отклонено"
	}
	actionDetails := "Решение: " + decisionText
	if req.Reason != "" {
		reasonText := getReasonText(req.Reason)
		actionDetails += ". Причина: " + reasonText
	}
	if req.Comment != "" {
		actionDetails += ". Комментарий: " + req.Comment
	}
	application.AddActionToHistory("decision_made", username, actionDetails)

	// Save the updated application
	result, err := h.service.UpdateApplication(id, application)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Decision made successfully",
		"application": result,
		"decision":    req.Decision,
		"reason":      req.Reason,
	})
}

// DocumentsRequestRequest represents a request for additional documents
type DocumentsRequestRequest struct {
	Documents []string `json:"documents" binding:"required"`
	Comment   string   `json:"comment,omitempty"`
}

// Helper function to get Russian text for rejection reasons
func getReasonText(reason string) string {
	reasonMap := map[string]string{
		"low_income":             "Недостаточный доход",
		"bad_credit_history":     "Плохая кредитная история",
		"high_debt":              "Высокий уровень задолженности",
		"insufficient_documents": "Недостаточно документов",
		"other":                  "Другое",
	}
	if text, ok := reasonMap[reason]; ok {
		return text
	}
	return reason
}

func approvedAmountExpr() string {
	return "COALESCE(SUM(requested_amount), 0)"
}

func avgProcessingMinutes(db *gorm.DB, query string, args ...interface{}) float64 {
	var applications []models.CreditApplication
	if err := db.Model(&models.CreditApplication{}).
		Where("review_started_at IS NOT NULL AND review_completed_at IS NOT NULL").
		Where(query, args...).
		Find(&applications).Error; err != nil {
		return 0
	}

	var totalMinutes float64
	var counted int
	for _, app := range applications {
		if app.ReviewStartedAt == nil || app.ReviewCompletedAt == nil {
			continue
		}
		duration := app.ReviewCompletedAt.Sub(*app.ReviewStartedAt)
		if duration <= 0 && !app.CreatedAt.IsZero() {
			duration = app.ReviewCompletedAt.Sub(app.CreatedAt)
		}
		if duration < 0 {
			continue
		}
		totalMinutes += duration.Minutes()
		counted++
	}
	if counted == 0 {
		return 0
	}
	return totalMinutes / float64(counted)
}

func periodRange(period string, now time.Time) (time.Time, time.Time) {
	location := now.Location()
	switch period {
	case "lastMonth":
		start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, location)
		return start, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	case "quarter":
		month := (int(now.Month())-1)/3*3 + 1
		start := time.Date(now.Year(), time.Month(month), 1, 0, 0, 0, 0, location)
		return start, start.AddDate(0, 3, 0)
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, location)
		return start, start.AddDate(1, 0, 0)
	default:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
		return start, start.AddDate(0, 1, 0)
	}
}

func previousPeriodRange(start, end time.Time) (time.Time, time.Time) {
	duration := end.Sub(start)
	return start.Add(-duration), start
}

func percentChange(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return (current - previous) * 100 / previous
}

// GetReportsData returns data for the reports dashboard
func (h *AdminHandler) GetReportsData(c *gin.Context) {
	db := h.service.GetDB()

	var reportsData struct {
		TotalApplications int64   `json:"totalApplications"`
		Approved          int64   `json:"approved"`
		Rejected          int64   `json:"rejected"`
		New               int64   `json:"new"`
		InReview          int64   `json:"inReview"`
		ManualReview      int64   `json:"manualReview"`
		ApprovedAmount    float64 `json:"approvedAmount"`
		AvgProcessingTime float64 `json:"avgProcessingTime"`
		MonthlyStats      []struct {
			Month          string  `json:"month"`
			Applications   int64   `json:"applications"`
			Approved       int64   `json:"approved"`
			Rejected       int64   `json:"rejected"`
			ProcessingTime float64 `json:"processingTime"`
		} `json:"monthlyStats"`
		StatusDistribution []struct {
			Status string `json:"status"`
			Count  int64  `json:"count"`
			Color  string `json:"color"`
		} `json:"statusDistribution"`
		CreditTypes []struct {
			Type       string  `json:"type"`
			Count      int64   `json:"count"`
			Amount     float64 `json:"amount"`
			Percentage float64 `json:"percentage"`
		} `json:"creditTypes"`
		Analysts []struct {
			Name         string  `json:"name"`
			Applications int64   `json:"applications"`
			Approved     int64   `json:"approved"`
			Rejected     int64   `json:"rejected"`
			AvgTime      float64 `json:"avgTime"`
			Rating       float64 `json:"rating"`
		} `json:"analysts"`
		RiskDistribution []struct {
			Level string `json:"level"`
			Count int64  `json:"count"`
			Color string `json:"color"`
		} `json:"riskDistribution"`
		ApprovalRate []struct {
			Month string  `json:"month"`
			Rate  float64 `json:"rate"`
		} `json:"approvalRate"`
		ProcessingTime []struct {
			Month string  `json:"month"`
			Time  float64 `json:"time"`
		} `json:"processingTime"`
		ProblematicApplications []struct {
			ID     string  `json:"id"`
			Client string  `json:"client"`
			Amount float64 `json:"amount"`
			Risk   string  `json:"risk"`
			Issue  string  `json:"issue"`
			Date   string  `json:"date"`
		} `json:"problematicApplications"`
	}

	// Get total counts
	db.Model(&models.CreditApplication{}).Count(&reportsData.TotalApplications)
	db.Model(&models.CreditApplication{}).Where("status = ?", "approved").Count(&reportsData.Approved)
	db.Model(&models.CreditApplication{}).Where("status = ?", "rejected").Count(&reportsData.Rejected)
	db.Model(&models.CreditApplication{}).Where("status = ?", "new").Count(&reportsData.New)
	db.Model(&models.CreditApplication{}).Where("status = ?", "in_review").Count(&reportsData.InReview)
	db.Model(&models.CreditApplication{}).Where("status = ?", "manual_review").Count(&reportsData.ManualReview)

	// Calculate approved amount
	var totalApproved float64
	db.Model(&models.CreditApplication{}).Where("status = ?", "approved").Select(approvedAmountExpr()).Scan(&totalApproved)
	reportsData.ApprovedAmount = totalApproved

	// Calculate average processing time
	reportsData.AvgProcessingTime = avgProcessingMinutes(db, "1 = 1")

	// Status distribution
	var statuses []struct {
		Status string
		Count  int64
	}
	db.Model(&models.CreditApplication{}).Select("status, COUNT(*) as count").Group("status").Scan(&statuses)
	statusNames := map[string]string{
		"new":                 "Новые",
		"in_review":           "В работе",
		"approved":            "Одобрено",
		"contract_sent":       "На подписании",
		"issued":              "Кредит выдан",
		"rejected":            "Отклонено",
		"manual_review":       "Ручная проверка",
		"pending":             "Ожидает",
		"documents_requested": "Запрошены документы",
		"revision":            "На доработке",
	}
	statusColors := map[string]string{
		"new":                 "#17a2b8",
		"in_review":           "#ffc107",
		"approved":            "#28a745",
		"contract_sent":       "#6f42c1",
		"issued":              "#198754",
		"rejected":            "#dc3545",
		"manual_review":       "#6f42c1",
		"pending":             "#6c757d",
		"documents_requested": "#fd7e14",
		"revision":            "#20c997",
	}

	reportsData.StatusDistribution = make([]struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
		Color  string `json:"color"`
	}, len(statuses))
	for i, status := range statuses {
		name := statusNames[status.Status]
		if name == "" {
			name = status.Status
		}
		color := statusColors[status.Status]
		if color == "" {
			color = "#6c757d"
		}
		reportsData.StatusDistribution[i] = struct {
			Status string `json:"status"`
			Count  int64  `json:"count"`
			Color  string `json:"color"`
		}{
			Status: name,
			Count:  status.Count,
			Color:  color,
		}
	}

	// Credit types
	var creditTypes []struct {
		CreditType     string
		Count          int64
		ApprovedAmount float64
	}
	db.Model(&models.CreditApplication{}).
		Select("credit_type as credit_type, COUNT(*) as count, "+approvedAmountExpr()+" as approved_amount").
		Where("status = ?", "approved").
		Group("credit_type").
		Scan(&creditTypes)

	var totalCount int64
	for _, ct := range creditTypes {
		totalCount += ct.Count
	}

	reportsData.CreditTypes = make([]struct {
		Type       string  `json:"type"`
		Count      int64   `json:"count"`
		Amount     float64 `json:"amount"`
		Percentage float64 `json:"percentage"`
	}, len(creditTypes))

	creditTypeMap := map[string]string{
		"ПК": "Потребительский",
		"АВ": "Автокредит",
		"АК": "Автокредит",
		"ИП": "Ипотека",
		"КК": "Кредитная карта",
		"БК": "Бизнес-кредит",
	}

	for i, ct := range creditTypes {
		typeName := creditTypeMap[ct.CreditType]
		if typeName == "" {
			typeName = ct.CreditType
		}
		percentage := 0.0
		if totalCount > 0 {
			percentage = float64(ct.Count) * 100.0 / float64(totalCount)
		}
		reportsData.CreditTypes[i] = struct {
			Type       string  `json:"type"`
			Count      int64   `json:"count"`
			Amount     float64 `json:"amount"`
			Percentage float64 `json:"percentage"`
		}{
			Type:       typeName,
			Count:      ct.Count,
			Amount:     ct.ApprovedAmount,
			Percentage: percentage,
		}
	}

	// Monthly stats (last 6 months) - calculate from database
	monthNames := []string{"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}
	now := time.Now()
	reportsData.MonthlyStats = make([]struct {
		Month          string  `json:"month"`
		Applications   int64   `json:"applications"`
		Approved       int64   `json:"approved"`
		Rejected       int64   `json:"rejected"`
		ProcessingTime float64 `json:"processingTime"`
	}, 6)

	for i := 5; i >= 0; i-- {
		month := now.AddDate(0, -i, 1)
		monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, now.Location())
		monthEnd := time.Date(month.Year(), month.Month()+1, 1, 0, 0, 0, 0, now.Location())

		var totalApps, approved, rejected int64
		db.Model(&models.CreditApplication{}).Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).Count(&totalApps)
		db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "approved", monthStart, monthEnd).Count(&approved)
		db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "rejected", monthStart, monthEnd).Count(&rejected)

		avgTime := avgProcessingMinutes(db, "review_completed_at >= ? AND review_completed_at < ?", monthStart, monthEnd)

		reportsData.MonthlyStats[5-i] = struct {
			Month          string  `json:"month"`
			Applications   int64   `json:"applications"`
			Approved       int64   `json:"approved"`
			Rejected       int64   `json:"rejected"`
			ProcessingTime float64 `json:"processingTime"`
		}{
			Month:          monthNames[month.Month()-1],
			Applications:   totalApps,
			Approved:       approved,
			Rejected:       rejected,
			ProcessingTime: avgTime,
		}
	}

	// Analysts (from database if available, otherwise default)
	var analysts []struct {
		AnalystName   string
		Count         int64
		ApprovedCount int64
		RejectedCount int64
	}
	db.Model(&models.CreditApplication{}).Select("analyst_name as analyst_name, COUNT(*) as count, SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) as approved_count, SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected_count").Group("analyst_name").Where("analyst_name != ''").Scan(&analysts)

	if len(analysts) > 0 {
		reportsData.Analysts = make([]struct {
			Name         string  `json:"name"`
			Applications int64   `json:"applications"`
			Approved     int64   `json:"approved"`
			Rejected     int64   `json:"rejected"`
			AvgTime      float64 `json:"avgTime"`
			Rating       float64 `json:"rating"`
		}, len(analysts))
		for i, a := range analysts {
			// Calculate average processing time for this analyst
			avgTime := avgProcessingMinutes(db, "analyst_name = ?", a.AnalystName)

			// Calculate rating based on approval rate
			rating := 4.0
			if a.Count > 0 {
				approvalRate := float64(a.ApprovedCount) * 100.0 / float64(a.Count)
				rating = 3.0 + (approvalRate/100.0)*2.0
				if rating > 5.0 {
					rating = 5.0
				}
			}

			reportsData.Analysts[i] = struct {
				Name         string  `json:"name"`
				Applications int64   `json:"applications"`
				Approved     int64   `json:"approved"`
				Rejected     int64   `json:"rejected"`
				AvgTime      float64 `json:"avgTime"`
				Rating       float64 `json:"rating"`
			}{
				Name:         a.AnalystName,
				Applications: a.Count,
				Approved:     a.ApprovedCount,
				Rejected:     a.RejectedCount,
				AvgTime:      avgTime,
				Rating:       rating,
			}
		}
	} else {
		reportsData.Analysts = []struct {
			Name         string  `json:"name"`
			Applications int64   `json:"applications"`
			Approved     int64   `json:"approved"`
			Rejected     int64   `json:"rejected"`
			AvgTime      float64 `json:"avgTime"`
			Rating       float64 `json:"rating"`
		}{}
	}

	// Risk distribution based on risk_score
	var riskLow, riskMedium, riskHigh, riskCritical int64
	db.Model(&models.CreditApplication{}).Where("risk_score < 25").Count(&riskLow)
	db.Model(&models.CreditApplication{}).Where("risk_score >= 25 AND risk_score < 50").Count(&riskMedium)
	db.Model(&models.CreditApplication{}).Where("risk_score >= 50 AND risk_score < 75").Count(&riskHigh)
	db.Model(&models.CreditApplication{}).Where("risk_score >= 75").Count(&riskCritical)

	reportsData.RiskDistribution = []struct {
		Level string `json:"level"`
		Count int64  `json:"count"`
		Color string `json:"color"`
	}{
		{Level: "Низкий", Count: riskLow, Color: "#28a745"},
		{Level: "Средний", Count: riskMedium, Color: "#ffc107"},
		{Level: "Высокий", Count: riskHigh, Color: "#fd7e14"},
		{Level: "Критический", Count: riskCritical, Color: "#dc3545"},
	}

	// Approval rate by month
	reportsData.ApprovalRate = make([]struct {
		Month string  `json:"month"`
		Rate  float64 `json:"rate"`
	}, 6)
	reportsData.ProcessingTime = make([]struct {
		Month string  `json:"month"`
		Time  float64 `json:"time"`
	}, 6)

	for i := 5; i >= 0; i-- {
		month := now.AddDate(0, -i, 1)
		monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, now.Location())
		monthEnd := time.Date(month.Year(), month.Month()+1, 1, 0, 0, 0, 0, now.Location())

		var totalApps, approved int64
		db.Model(&models.CreditApplication{}).Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).Count(&totalApps)
		db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "approved", monthStart, monthEnd).Count(&approved)

		rate := 0.0
		if totalApps > 0 {
			rate = float64(approved) * 100.0 / float64(totalApps)
		}

		avgTime := avgProcessingMinutes(db, "review_completed_at >= ? AND review_completed_at < ?", monthStart, monthEnd)

		reportsData.ApprovalRate[5-i] = struct {
			Month string  `json:"month"`
			Rate  float64 `json:"rate"`
		}{
			Month: monthNames[month.Month()-1],
			Rate:  rate,
		}

		reportsData.ProcessingTime[5-i] = struct {
			Month string  `json:"month"`
			Time  float64 `json:"time"`
		}{
			Month: monthNames[month.Month()-1],
			Time:  avgTime,
		}
	}

	// Problematic applications (high risk or rejected)
	var problems []struct {
		ID              string
		ClientName      string
		RequestedAmount float64
		RiskScore       int
		Status          string
		DecisionReason  string
		CreatedAt       time.Time
	}
	db.Model(&models.CreditApplication{}).Select("id, client_name, requested_amount, risk_score, status, decision_reason, created_at").Where("risk_score >= 50 OR status = 'rejected'").Order("risk_score DESC").Limit(10).Scan(&problems)

	reportsData.ProblematicApplications = make([]struct {
		ID     string  `json:"id"`
		Client string  `json:"client"`
		Amount float64 `json:"amount"`
		Risk   string  `json:"risk"`
		Issue  string  `json:"issue"`
		Date   string  `json:"date"`
	}, len(problems))

	issues := map[string]string{
		"rejected":     "Заявка отклонена",
		"high_risk":    "Высокий уровень риска",
		"low_credit":   "Низкий кредитный рейтинг",
		"high_dti":     "Высокая закредитованность",
		"income_issue": "Проблемы с доходом",
	}

	for i, p := range problems {
		riskLevel := "Средний"
		if p.RiskScore >= 75 {
			riskLevel = "Критический"
		} else if p.RiskScore >= 50 {
			riskLevel = "Высокий"
		} else if p.RiskScore < 25 {
			riskLevel = "Низкий"
		}

		issue := p.DecisionReason
		if issue == "" {
			if p.Status == "rejected" {
				issue = issues["rejected"]
			} else {
				issue = issues["high_risk"]
			}
		}

		reportsData.ProblematicApplications[i] = struct {
			ID     string  `json:"id"`
			Client string  `json:"client"`
			Amount float64 `json:"amount"`
			Risk   string  `json:"risk"`
			Issue  string  `json:"issue"`
			Date   string  `json:"date"`
		}{
			ID:     p.ID,
			Client: p.ClientName,
			Amount: p.RequestedAmount,
			Risk:   riskLevel,
			Issue:  issue,
			Date:   p.CreatedAt.Format("2006-01-02"),
		}
	}

	c.JSON(http.StatusOK, reportsData)
}

// GetMetricsByPeriod returns metrics for a specific period
func (h *AdminHandler) GetMetricsByPeriod(c *gin.Context) {
	period := c.Query("period")
	if period == "" {
		period = "currentMonth"
	}

	db := h.service.GetDB()

	var metrics struct {
		TotalApplications int64   `json:"totalApplications"`
		Approved          int64   `json:"approved"`
		ApprovedAmount    float64 `json:"approvedAmount"`
		AvgProcessingTime float64 `json:"avgProcessingTime"`
		ApprovalRate      float64 `json:"approvalRate"`
		Changes           struct {
			TotalApplications float64 `json:"totalApplications"`
			Approved          float64 `json:"approved"`
			ApprovedAmount    float64 `json:"approvedAmount"`
			AvgProcessingTime float64 `json:"avgProcessingTime"`
			ApprovalRate      float64 `json:"approvalRate"`
		} `json:"changes"`
	}

	now := time.Now()
	startDate, endDate := periodRange(period, now)
	prevStartDate, prevEndDate := previousPeriodRange(startDate, endDate)

	db.Model(&models.CreditApplication{}).Where("created_at >= ? AND created_at < ?", startDate, endDate).Count(&metrics.TotalApplications)
	db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "approved", startDate, endDate).Count(&metrics.Approved)

	var totalAmount float64
	db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "approved", startDate, endDate).Select(approvedAmountExpr()).Scan(&totalAmount)
	metrics.ApprovedAmount = totalAmount

	metrics.ApprovalRate = 0
	if metrics.TotalApplications > 0 {
		metrics.ApprovalRate = float64(metrics.Approved) * 100.0 / float64(metrics.TotalApplications)
	}

	metrics.AvgProcessingTime = avgProcessingMinutes(db, "review_completed_at >= ? AND review_completed_at < ?", startDate, endDate)

	var prevTotalApplications, prevApproved int64
	var prevApprovedAmount float64
	db.Model(&models.CreditApplication{}).Where("created_at >= ? AND created_at < ?", prevStartDate, prevEndDate).Count(&prevTotalApplications)
	db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "approved", prevStartDate, prevEndDate).Count(&prevApproved)
	db.Model(&models.CreditApplication{}).Where("status = ? AND created_at >= ? AND created_at < ?", "approved", prevStartDate, prevEndDate).Select(approvedAmountExpr()).Scan(&prevApprovedAmount)

	prevApprovalRate := 0.0
	if prevTotalApplications > 0 {
		prevApprovalRate = float64(prevApproved) * 100.0 / float64(prevTotalApplications)
	}
	prevAvgProcessingTime := avgProcessingMinutes(db, "review_completed_at >= ? AND review_completed_at < ?", prevStartDate, prevEndDate)

	metrics.Changes.TotalApplications = percentChange(float64(metrics.TotalApplications), float64(prevTotalApplications))
	metrics.Changes.Approved = percentChange(float64(metrics.Approved), float64(prevApproved))
	metrics.Changes.ApprovedAmount = percentChange(metrics.ApprovedAmount, prevApprovedAmount)
	metrics.Changes.AvgProcessingTime = percentChange(metrics.AvgProcessingTime, prevAvgProcessingTime)
	metrics.Changes.ApprovalRate = metrics.ApprovalRate - prevApprovalRate

	c.JSON(http.StatusOK, metrics)
}

// RequestDocuments handles requesting additional documents from the applicant
func (h *AdminHandler) RequestDocuments(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var req DocumentsRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Documents) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one document must be specified"})
		return
	}

	// Get username from context
	username := c.GetString("username")
	if username == "" {
		username = "Администратор"
	}

	// Get the existing application
	application, err := h.service.GetApplicationByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Only allow document requests for applications in review
	if application.Status != "in_review" && application.Status != "new" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only request documents for applications in review"})
		return
	}

	// Update the application status to revision
	application.Status = "revision"

	// Add action to history
	documentsStr := ""
	for i, doc := range req.Documents {
		if i > 0 {
			documentsStr += ", "
		}
		documentsStr += doc
	}
	actionDetails := "Запрос документов: " + documentsStr
	if req.Comment != "" {
		actionDetails += ". Комментарий: " + req.Comment
	}
	application.AddActionToHistory("documents_requested", username, actionDetails)

	// Save the updated application
	result, err := h.service.UpdateApplication(id, application)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Documents requested successfully",
		"application": result,
		"documents":   req.Documents,
	})
}

// AIRecommendationRequest represents a request for AI recommendation
type AIRecommendationRequest struct {
	// No body needed, all info comes from the application
}

// GenerateAIRecommendation builds a recommendation letter from BKI data.
func (h *AdminHandler) GenerateAIRecommendation(c *gin.Context) {
	appID := c.Param("id")
	db := h.service.GetDB()

	var app models.CreditApplication
	if err := db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	dti := app.DTIRatio
	if dti == 0 {
		dti = app.DebtBurdenRatio
	}

	currentDelinquency := app.CurrentDelinquencies
	maxDelinquencyDays := app.MaxDelinquencyDays
	totalDebt := app.TotalDebt
	var activeCredits []map[string]interface{}
	if app.ActiveCreditsList != "" {
		_ = json.Unmarshal([]byte(app.ActiveCreditsList), &activeCredits)
		for _, credit := range activeCredits {
			if debt, ok := credit["debt"].(float64); ok && totalDebt == 0 {
				totalDebt += debt
			}
			if days, ok := credit["days_overdue"].(float64); ok {
				if int(days) > maxDelinquencyDays {
					maxDelinquencyDays = int(days)
				}
				if days > 0 {
					currentDelinquency = true
				}
			}
			if status, ok := credit["payment_status"].(string); ok && (status == "delayed" || status == "critical") {
				currentDelinquency = true
			}
		}
	}

	creditComponent := (app.CreditScore - 300) * 40 / 550
	if creditComponent < 0 {
		creditComponent = 0
	}
	if creditComponent > 40 {
		creditComponent = 40
	}

	paymentComponent := 25
	if app.DelayedPayments12m > 0 {
		paymentComponent -= app.DelayedPayments12m * 3
	}
	if currentDelinquency {
		paymentComponent -= 10
	}
	if maxDelinquencyDays > 90 {
		paymentComponent -= 10
	}
	if paymentComponent < 0 {
		paymentComponent = 0
	}

	debtComponent := 20
	if dti > 0.6 {
		debtComponent = 0
	} else if dti > 0.5 {
		debtComponent = 5
	} else if dti > 0.4 {
		debtComponent = 10
	} else if dti > 0.3 {
		debtComponent = 15
	}

	incomeComponent := 5
	if app.MonthlyIncome <= 0 || (app.RequestedAmount > 0 && app.MonthlyIncome > 0 && app.RequestedAmount/app.MonthlyIncome > 12) {
		incomeComponent = 0
	}

	score := creditComponent + paymentComponent + debtComponent + incomeComponent + 10
	if score > 100 {
		score = 100
	}

	hardStopReasons := []string{}
	if app.CreditScore < 550 {
		hardStopReasons = append(hardStopReasons, "кредитный рейтинг ниже 550")
	}
	if currentDelinquency {
		hardStopReasons = append(hardStopReasons, "есть текущие просрочки по кредитам")
	}
	if maxDelinquencyDays > 90 {
		hardStopReasons = append(hardStopReasons, "были просрочки свыше 90 дней")
	}
	if app.DelayedPayments12m >= 6 {
		hardStopReasons = append(hardStopReasons, "6 и более просрочек за последние 12 месяцев")
	}
	if dti > 0.6 {
		hardStopReasons = append(hardStopReasons, "долговая нагрузка превышает 60% дохода")
	}

	recommendation := "reject"
	decisionText := "не одобрять заявку"
	if score >= 70 && len(hardStopReasons) == 0 {
		recommendation = "approve"
		decisionText = "одобрить заявку"
	}

	letterParts := []string{
		fmt.Sprintf("БКИ-анализ: рейтинг %d, активных кредитов %d, просрочек за 12 месяцев %d, максимальная просрочка %d дней, долговая нагрузка %.1f%%.",
			app.CreditScore, app.ActiveCredits, app.DelayedPayments12m, maxDelinquencyDays, dti*100),
	}
	if len(hardStopReasons) > 0 {
		letterParts = append(letterParts, "Ключевые риски: "+strings.Join(hardStopReasons, "; ")+".")
	} else {
		letterParts = append(letterParts, "Критических просрочек не выявлено, долговая нагрузка допустимая, профиль БКИ соответствует минимальным требованиям.")
	}
	letterParts = append(letterParts, fmt.Sprintf("Вывод: %s.", decisionText))

	comment := strings.Join(letterParts, "\n\n")

	activeCreditsValue := 100 - app.ActiveCredits*8
	if activeCreditsValue < 0 {
		activeCreditsValue = 0
	}

	factors := []map[string]interface{}{
		{"text": fmt.Sprintf("Кредитный рейтинг БКИ (%d)", app.CreditScore), "value": creditComponent * 100 / 40, "type": "credit"},
		{"text": fmt.Sprintf("Платёжная дисциплина: %d просрочек за 12 месяцев", app.DelayedPayments12m), "value": paymentComponent * 100 / 25, "type": "payment"},
		{"text": fmt.Sprintf("Долговая нагрузка %.1f%%", dti*100), "value": debtComponent * 100 / 20, "type": "debt"},
		{"text": fmt.Sprintf("Активные кредиты: %d, общий долг: %.0f ₽", app.ActiveCredits, totalDebt), "value": activeCreditsValue, "type": "active_credits"},
		{"text": fmt.Sprintf("Доход и сумма кредита: %.0f ₽ / %.0f ₽", app.MonthlyIncome, app.RequestedAmount), "value": incomeComponent * 100 / 5, "type": "income"},
	}

	factorsJSON, _ := json.Marshal(factors)

	// Store the generated letter in existing recommendation fields.
	app.AIScore = score
	app.AIRecommendation = recommendation
	app.AIComment = comment
	app.RecommendationReason = "recommendation_letter"
	app.RiskScore = 100 - score
	app.FactorsAnalysis = string(factorsJSON)
	app.CurrentDelinquencies = currentDelinquency
	app.MaxDelinquencyDays = maxDelinquencyDays
	app.TotalDebt = totalDebt

	var positiveFactors []string
	var riskFactors []string

	if app.CreditScore >= 700 {
		positiveFactors = append(positiveFactors, "Высокий кредитный рейтинг")
	} else if app.CreditScore >= 600 {
		positiveFactors = append(positiveFactors, "Удовлетворительный кредитный рейтинг")
	} else {
		riskFactors = append(riskFactors, "Низкий кредитный рейтинг")
	}

	if dti <= 0.4 {
		positiveFactors = append(positiveFactors, "Допустимая долговая нагрузка")
	} else if dti > 0.5 {
		riskFactors = append(riskFactors, "Высокая долговая нагрузка")
	}
	if app.DelayedPayments12m > 0 {
		riskFactors = append(riskFactors, fmt.Sprintf("Просрочки за 12 месяцев: %d", app.DelayedPayments12m))
	}
	if currentDelinquency {
		riskFactors = append(riskFactors, "Есть текущие просрочки")
	}

	if len(positiveFactors) == 0 {
		positiveFactors = []string{"Достаточные данные для анализа"}
	}
	if len(riskFactors) == 0 {
		riskFactors = []string{"Требуется дополнительная проверка"}
	}

	app.PositiveFactors = strings.Join(positiveFactors, "; ")
	app.RiskFactors = strings.Join(riskFactors, "; ")

	// Save updated application
	if err := db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add action to history
	app.AddActionToHistory("recommendation_letter", "Система", fmt.Sprintf("Сформировано рекомендательное письмо. Решение: %s. Балл БКИ: %d", decisionText, score))
	db.Save(&app)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Рекомендательное письмо сформировано",
		"recommendation": recommendation,
		"comment":        comment,
		"score":          score,
		"risk_score":     app.RiskScore,
		"credit_score":   app.CreditScore,
		"factors":        string(factorsJSON),
		"letter_ready":   true,
	})
}
