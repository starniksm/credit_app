package routes

import (
	"net/http"

	"credit_app/handlers"
	"credit_app/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	// Serve static files
	r.Static("/static", "./static")

	// Initialize handlers with database
	appHandler := handlers.NewApplicationHandler(db)
	adminHandler := handlers.NewAdminHandler(db)
	repHandler := handlers.NewRepresentativeHandler(db)
	docHandler := handlers.NewDocumentHandler(db)
	adminUserHandler := handlers.NewAdminUserHandler(db)
	creditManagerHandler := handlers.NewCreditManagerHandler(db)

	// Define routes
	api := r.Group("/api")
	{
		// Public endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Login(db))
			auth.POST("/register", handlers.Register(db))
		}

		contracts := api.Group("/contracts")
		{
			contracts.POST("/:id/signed", docHandler.MarkContractSigned)
		}

		credits := api.Group("/credits")
		{
			credits.POST("/delinquency", docHandler.MarkCreditDelinquency)
			credits.POST("/restructure/decision", docHandler.MarkRestructureDecision)
		}

		// Protected endpoints
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			applications := protected.Group("/applications")
			{
				applications.POST("/", appHandler.CreateApplication)
				applications.GET("/", appHandler.GetApplications)
				applications.GET("/:id", appHandler.GetApplicationByID)
				applications.PUT("/:id", appHandler.UpdateApplication)
				applications.DELETE("/:id", middleware.RoleMiddleware("admin"), appHandler.DeleteApplication)

				// BKI endpoints
				applications.POST("/:id/request-bki", appHandler.RequestBKIData)
			}

			// Admin/Analyst-specific endpoints
			admin := protected.Group("/admin")
			admin.Use(middleware.AnalystOnly())
			{
				admin.GET("/dashboard/stats", adminHandler.GetDashboardStats)
				admin.GET("/reports/data", adminHandler.GetReportsData)
				admin.GET("/reports/metrics", adminHandler.GetMetricsByPeriod)
				admin.POST("/applications/:id/start-review", adminHandler.StartReview)
				admin.PUT("/applications/:id/status", adminHandler.UpdateApplicationStatus)
				admin.GET("/applications/export", adminHandler.ExportApplications)
				admin.GET("/applications/review", adminHandler.GetApplicationsForReview)
				// Decision endpoints
				admin.POST("/applications/:id/decision", adminHandler.MakeDecision)
				admin.POST("/applications/:id/documents-request", adminHandler.RequestDocuments)
				// AI recommendation endpoint
				admin.POST("/applications/:id/ai-recommendation", adminHandler.GenerateAIRecommendation)
			}

			// Admin panel endpoints (admin only)
			adminPanel := protected.Group("/admin-panel")
			adminPanel.Use(middleware.RoleMiddleware("admin"))
			{
				adminPanel.GET("/users", adminUserHandler.GetUsers)
				adminPanel.POST("/users", adminUserHandler.CreateUser)
				adminPanel.PUT("/users/:id", adminUserHandler.UpdateUser)
				adminPanel.PUT("/users/:id/password", adminUserHandler.ChangeUserPassword)
				adminPanel.PUT("/users/:id/role", adminUserHandler.UpdateUserRole)
				adminPanel.DELETE("/users/:id", adminUserHandler.DeleteUser)
			}

			// Representative-specific endpoints
			representative := protected.Group("/representative")
			representative.Use(middleware.RepresentativeOnly())
			{
				representative.GET("/clients", repHandler.GetClients)
				representative.GET("/clients/:id", repHandler.GetClientByID)
				representative.GET("/meetings", repHandler.GetMeetings)
				representative.POST("/meetings", repHandler.CreateMeeting)
				representative.PUT("/meetings/:id/status", repHandler.UpdateMeetingStatus)
				representative.POST("/applications", repHandler.CreateCardApplication)
			}

			// Credit manager endpoints
			creditManager := protected.Group("/credit-manager")
			creditManager.Use(middleware.CreditManagerOnly())
			{
				creditManager.GET("/clients", creditManagerHandler.GetProblemClients)
				creditManager.GET("/clients/:id/assessment", creditManagerHandler.GetClientAssessment)
				creditManager.POST("/clients/:id/income-request", creditManagerHandler.RequestOfficialIncomeInfo)
				creditManager.POST("/clients/:id/restructure", creditManagerHandler.RestructureClient)
			}

			// Document generation endpoints
			documents := protected.Group("/documents")
			{
				documents.GET("/:id/contract-pdf", docHandler.GenerateContractPDF)
				documents.GET("/:id/schedule-pdf", docHandler.GeneratePaymentSchedulePDF)
				documents.POST("/:id/send-to-client", docHandler.SendDocumentsToClient)
			}
		}
	}

	// Frontend routes
	// Role selection page - main entry point
	r.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/role-select.html")
	})

	r.GET("/role-select", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/role-select.html")
	})

	// Analyst dashboard
	r.GET("/analyst", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/admin.html")
	})

	// Admin panel
	r.GET("/admin-panel", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/admin-panel.html")
	})

	r.GET("/application/:id", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/application.html")
	})

	r.GET("/reports", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/reports.html")
	})

	// Credit process routes
	r.GET("/product-selection", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/product-selection.html")
	})

	r.GET("/contract-generation", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/contract-generation.html")
	})

	r.GET("/payment-schedule", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/payment-schedule.html")
	})

	// Representative routes
	r.GET("/representative", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Redirect(http.StatusFound, "/representative/clients")
	})

	r.GET("/representative/clients", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/representative/clients.html")
	})

	r.GET("/representative/schedule", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/representative/schedule.html")
	})

	r.GET("/representative/meeting-status", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/representative/meeting-status.html")
	})

	r.GET("/representative/new-application", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/representative/new-application.html")
	})

	// Credit manager routes
	r.GET("/credit-manager", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/credit-manager.html")
	})

	r.GET("/credit-manager/restructure/:id", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./static/credit-restructure.html")
	})

	// Health check endpoint (public)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})
}
