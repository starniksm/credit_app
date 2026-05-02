package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleMiddleware checks if the user has the required role
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		c.Abort()
	}
}

// AnalystOnly middleware - allows only analysts and admins
func AnalystOnly() gin.HandlerFunc {
	return RoleMiddleware("analyst", "admin")
}

// RepresentativeOnly middleware - allows representatives and admins
func RepresentativeOnly() gin.HandlerFunc {
	return RoleMiddleware("representative", "admin")
}
