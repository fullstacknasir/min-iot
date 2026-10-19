package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/config"
	"miniot/apps/api/internal/database"
	"miniot/apps/api/internal/handlers"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
)

func TestSecurityAndReliability(t *testing.T) {
	cfg := config.LoadConfig()
	dbClients, err := database.InitDatabase(cfg)
	if err != nil {
		t.Skipf("Skipping database integration tests: %v", err)
	}
	defer dbClients.Pool.Close()

	_ = database.RunMigrations(context.Background(), dbClients.Pool)

	t.Run("Reliability - Health Check Returns 200 When Healthy", func(t *testing.T) {
		router := gin.New()
		healthHandler := handlers.NewHealthHandler(dbClients)
		router.GET("/health", healthHandler.Check)

		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp models.SystemHealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, "connected", resp.Services["timescaledb"])
		assert.Equal(t, "connected", resp.Services["redis"])
	})

	t.Run("Reliability - Health Check Nil Safety Without Panic", func(t *testing.T) {
		router := gin.New()
		nilHealthHandler := handlers.NewHealthHandler(nil)
		router.GET("/health", nilHealthHandler.Check)

		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		assert.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("Security - APIRateLimiter Blocks Flooding", func(t *testing.T) {
		router := gin.New()
		// Allow 3 requests per 10 seconds for test
		router.GET("/test-limit", middleware.APIRateLimiter(dbClients.Redis, 3, 10*time.Second), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest("GET", "/test-limit", nil)
			req.RemoteAddr = "192.0.2.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 4th request must be rate limited
		req, _ := http.NewRequest("GET", "/test-limit", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.NotEmpty(t, w.Header().Get("Retry-After"))
	})
}
