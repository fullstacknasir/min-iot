package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"miniot/apps/api/internal/security"
)

const (
	HeaderAuthorization   = "Authorization"
	BearerPrefix          = "Bearer "
	ContextKeyUserID      = "user_id"
	ContextKeyOrgID       = "organization_id"
	ContextKeyUserRole    = "user_role"
	ContextKeyUserEmail   = "user_email"
)

// AuthMiddleware verifies the JWT access token and binds claims to Gin context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenString(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token required via Bearer header or token query parameter"})
			return
		}

		claims, err := security.ValidateAccessToken(jwtSecret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyOrgID, claims.OrganizationID)
		c.Set(ContextKeyUserRole, claims.Role)
		c.Set(ContextKeyUserEmail, claims.Email)
		c.Next()
	}
}

// RequireRole ensures that the authenticated caller has one of the required roles.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(ContextKeyUserRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		roleStr := userRole.(string)
		for _, allowed := range allowedRoles {
			if roleStr == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions for this operation",
			"required_roles": allowedRoles,
			"current_role": roleStr,
		})
	}
}

// GetOrganizationID extracts the current authenticated organization ID from context.
func GetOrganizationID(c *gin.Context) string {
	val, exists := c.Get(ContextKeyOrgID)
	if !exists {
		return ""
	}
	return val.(string)
}

// GetUserID extracts the current authenticated user ID from context.
func GetUserID(c *gin.Context) string {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return ""
	}
	return val.(string)
}

// GetUserRole extracts the authenticated user's role from context.
func GetUserRole(c *gin.Context) string {
	val, exists := c.Get(ContextKeyUserRole)
	if !exists {
		return ""
	}
	return val.(string)
}

func extractTokenString(c *gin.Context) string {
	authHeader := c.GetHeader(HeaderAuthorization)
	if authHeader != "" && strings.HasPrefix(authHeader, BearerPrefix) {
		return strings.TrimPrefix(authHeader, BearerPrefix)
	}
	return c.Query("token")
}
