package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/security"
)

const (
	TestJWTSecret = "jwt_secret_middleware_testing_key_999"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware(TestJWTSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": GetUserID(c),
			"org_id":  GetOrganizationID(c),
			"role":    GetUserRole(c),
		})
	})

	user := &models.User{
		ID:             "usr-valid",
		OrganizationID: "org-valid",
		Email:          "valid@test.io",
		Role:           models.RoleOperator,
	}

	token, _ := security.GenerateAccessToken(TestJWTSecret, user, 10*time.Minute)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "usr-valid")
	assert.Contains(t, w.Body.String(), "org-valid")
}

func TestAuthMiddlewareMissingAndInvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware(TestJWTSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Missing header
	req1, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// Malformed token
	req2, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestRequireRole(t *testing.T) {
	setupRoleRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(ContextKeyUserRole, c.GetHeader("X-Test-Role"))
			c.Next()
		})
		r.POST("/admin-only", RequireRole(models.RoleAdmin), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r
	}

	r := setupRoleRouter()

	// Admin call should succeed
	reqAdmin, _ := http.NewRequest(http.MethodPost, "/admin-only", nil)
	reqAdmin.Header.Set("X-Test-Role", models.RoleAdmin)
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)
	assert.Equal(t, http.StatusOK, wAdmin.Code)

	// Viewer call should be forbidden
	reqViewer, _ := http.NewRequest(http.MethodPost, "/admin-only", nil)
	reqViewer.Header.Set("X-Test-Role", models.RoleViewer)
	wViewer := httptest.NewRecorder()
	r.ServeHTTP(wViewer, reqViewer)
	assert.Equal(t, http.StatusForbidden, wViewer.Code)
}

func TestLoginRateLimiter(t *testing.T) {
	router := gin.New()
	// Allow only 2 attempts per 10 seconds
	router.POST("/login", LoginRateLimiter(nil, 2, 10*time.Second), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for attempt := 1; attempt <= 2; attempt++ {
		req, _ := http.NewRequest(http.MethodPost, "/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Attempt %d should succeed", attempt)
	}

	// 3rd attempt exceeds limit
	reqBlocked, _ := http.NewRequest(http.MethodPost, "/login", nil)
	wBlocked := httptest.NewRecorder()
	router.ServeHTTP(wBlocked, reqBlocked)
	assert.Equal(t, http.StatusTooManyRequests, wBlocked.Code, "3rd attempt must be rate-limited")
	assert.Contains(t, wBlocked.Body.String(), "too many login attempts")
}
