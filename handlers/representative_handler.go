package handlers

import (
	"net/http"
	"time"

	"credit_app/models"
	"credit_app/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RepresentativeHandler struct {
	service *services.ApplicationService
}

func NewRepresentativeHandler(db *gorm.DB) *RepresentativeHandler {
	return &RepresentativeHandler{
		service: services.NewApplicationService(db),
	}
}

type ClientResponse struct {
	ID             string    `json:"id"`
	ClientID       string    `json:"clientId"`
	FullName       string    `json:"fullName"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	ApprovedAmount float64   `json:"approvedAmount"`
	CreditType     string    `json:"creditType"`
	ApprovedAt     time.Time `json:"approvedAt"`
	Priority       string    `json:"priority"`
	ContactStatus  string    `json:"contactStatus"`
	AssignedTo     string    `json:"assignedTo"`
}

func (h *RepresentativeHandler) GetClients(c *gin.Context) {
	db := h.service.GetDB()

	var clients []models.CreditApplication
	db.Where("status = ? AND card_issued = ?", "approved", false).Find(&clients)

	result := make([]ClientResponse, len(clients))
	for i, app := range clients {
		priority := "medium"
		contactStatus := "pending"

		daysSinceApproval := int(time.Since(app.CreatedAt).Hours() / 24)
		if daysSinceApproval > 14 {
			priority = "high"
		} else if daysSinceApproval < 3 {
			priority = "low"
		}

		result[i] = ClientResponse{
			ID:             app.ID,
			ClientID:       app.ID,
			FullName:       app.ClientName,
			Phone:          app.Phone,
			Email:          app.Email,
			ApprovedAmount: app.ApprovedAmount,
			CreditType:     app.CreditType,
			ApprovedAt:     app.CreatedAt,
			Priority:       priority,
			ContactStatus:  contactStatus,
			AssignedTo:     app.ReviewerID,
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *RepresentativeHandler) GetClientByID(c *gin.Context) {
	id := c.Param("id")
	db := h.service.GetDB()

	var app models.CreditApplication
	if err := db.Where("id = ? AND status = ?", id, "approved").First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := ClientResponse{
		ID:             app.ID,
		ClientID:       app.ID,
		FullName:       app.ClientName,
		Phone:          app.Phone,
		Email:          app.Email,
		ApprovedAmount: app.ApprovedAmount,
		CreditType:     app.CreditType,
		ApprovedAt:     app.CreatedAt,
	}

	c.JSON(http.StatusOK, client)
}

type Meeting struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"clientId"`
	ClientName  string    `json:"clientName"`
	ClientPhone string    `json:"clientPhone"`
	ClientEmail string    `json:"clientEmail"`
	Location    string    `json:"location"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Status      string    `json:"status"`
	Result      string    `json:"result"`
	CardType    string    `json:"cardType"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (h *RepresentativeHandler) GetMeetings(c *gin.Context) {
	db := h.service.GetDB()

	var meetings []models.Meeting
	db.Order("scheduled_at DESC").Find(&meetings)

	result := make([]Meeting, len(meetings))
	for i, m := range meetings {
		result[i] = Meeting{
			ID:          m.ID,
			ClientID:    m.ClientID,
			ClientName:  m.ClientName,
			ClientPhone: m.ClientPhone,
			ClientEmail: m.ClientEmail,
			Location:    m.Location,
			ScheduledAt: m.ScheduledAt,
			Status:      m.Status,
			Result:      m.Result,
			CardType:    m.CardType,
			Notes:       m.Notes,
			CreatedAt:   m.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, result)
}

type CreateMeetingRequest struct {
	ClientID    string    `json:"clientId" binding:"required"`
	ClientName  string    `json:"clientName" binding:"required"`
	ClientPhone string    `json:"clientPhone"`
	ClientEmail string    `json:"clientEmail"`
	Location    string    `json:"location"`
	ScheduledAt time.Time `json:"scheduledAt" binding:"required"`
}

func (h *RepresentativeHandler) CreateMeeting(c *gin.Context) {
	var req CreateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := h.service.GetDB()

	meeting := models.Meeting{
		ClientID:    req.ClientID,
		ClientName:  req.ClientName,
		ClientPhone: req.ClientPhone,
		ClientEmail: req.ClientEmail,
		Location:    req.Location,
		ScheduledAt: req.ScheduledAt,
		Status:      "scheduled",
	}

	if err := db.Create(&meeting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create meeting"})
		return
	}

	c.JSON(http.StatusCreated, Meeting{
		ID:          meeting.ID,
		ClientID:    meeting.ClientID,
		ClientName:  meeting.ClientName,
		ClientPhone: meeting.ClientPhone,
		ClientEmail: meeting.ClientEmail,
		Location:    meeting.Location,
		ScheduledAt: meeting.ScheduledAt,
		Status:      meeting.Status,
	})
}

type UpdateMeetingStatusRequest struct {
	Status   string `json:"status"`
	Result   string `json:"result"`
	CardType string `json:"cardType"`
	Notes    string `json:"notes"`
}

func (h *RepresentativeHandler) UpdateMeetingStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdateMeetingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := h.service.GetDB()

	var meeting models.Meeting
	if err := db.First(&meeting, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meeting not found"})
		return
	}

	meeting.Status = req.Status
	meeting.Result = req.Result
	meeting.CardType = req.CardType
	meeting.Notes = req.Notes

	if err := db.Save(&meeting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update meeting"})
		return
	}

	c.JSON(http.StatusOK, Meeting{
		ID:          meeting.ID,
		ClientID:    meeting.ClientID,
		ClientName:  meeting.ClientName,
		ClientPhone: meeting.ClientPhone,
		Location:    meeting.Location,
		ScheduledAt: meeting.ScheduledAt,
		Status:      meeting.Status,
		Result:      meeting.Result,
		CardType:    meeting.CardType,
		Notes:       meeting.Notes,
	})
}

type CardApplicationRequest struct {
	LastName            string  `json:"lastName"`
	FirstName           string  `json:"firstName"`
	MiddleName          string  `json:"middleName"`
	BirthDate           string  `json:"birthDate"`
	PassportSeries      string  `json:"passportSeries"`
	PassportNumber      string  `json:"passportNumber"`
	PassportIssueDate   string  `json:"passportIssueDate"`
	PassportIssuer      string  `json:"passportIssuer"`
	RegistrationAddress string  `json:"registrationAddress"`
	ResidenceAddress    string  `json:"residenceAddress"`
	Phone               string  `json:"phone"`
	Email               string  `json:"email"`
	EmploymentStatus    string  `json:"employmentStatus"`
	EmployerName        string  `json:"employerName"`
	Position            string  `json:"position"`
	WorkExperience      int     `json:"workExperience"`
	MonthlyIncome       float64 `json:"monthlyIncome"`
	CardType            string  `json:"cardType"`
	RelatedCreditID     string  `json:"relatedCreditId"`
}

func (h *RepresentativeHandler) CreateCardApplication(c *gin.Context) {
	var req CardApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := h.service.GetDB()

	application := models.CreditApplication{
		ClientName:          req.FirstName + " " + req.LastName,
		Phone:               req.Phone,
		Email:               req.Email,
		PassportSeries:      req.PassportSeries,
		PassportNumber:      req.PassportNumber,
		RegistrationAddress: req.RegistrationAddress,
		ResidenceAddress:    req.ResidenceAddress,
		CreditType:          "КК",
		Status:              "pending",
		EmploymentStatus:    req.EmploymentStatus,
		EmployerName:        req.EmployerName,
		Position:            req.Position,
		MonthlyIncome:       req.MonthlyIncome,
	}

	if err := db.Create(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Application created successfully",
		"application": application,
	})
}
