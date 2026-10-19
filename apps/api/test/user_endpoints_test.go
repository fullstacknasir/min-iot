package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/config"
	"miniot/apps/api/internal/database"
	"miniot/apps/api/internal/handlers"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
)

func setupUserTestRouter(t *testing.T) (*gin.Engine, *database.Clients, *config.Config) {
	cfg := config.LoadConfig()
	dbClients, err := database.InitDatabase(cfg)
	if err != nil {
		t.Skipf("Skipping database integration tests: %v", err)
	}

	_ = database.RunMigrations(context.Background(), dbClients.Pool)

	router := gin.New()
	router.Use(gin.Recovery())

	authRepo := repository.NewAuthRepository(dbClients.Pool)
	authHandler := handlers.NewAuthHandler(authRepo, cfg)
	userHandler := handlers.NewUserHandler(authRepo)

	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		users := api.Group("/users")
		users.Use(middleware.RequireRole(models.RoleAdmin))
		{
			users.GET("", userHandler.ListUsers)
			users.POST("", userHandler.CreateUser)
			users.PATCH("/:id/role", userHandler.UpdateUserRole)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	return router, dbClients, cfg
}

func registerAndGetToken(t *testing.T, router *gin.Engine, email, orgID, role string) string {
	reg := models.RegisterRequest{
		OrganizationID: orgID,
		Email:          email,
		Password:       "SecurePassword123!",
		Role:           role,
	}
	res := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", reg, "")
	return res.AccessToken
}


func TestUserManagementEndpointsAndScoping(t *testing.T) {
	router, dbClients, _ := setupUserTestRouter(t)
	if router == nil {
		return
	}
	defer dbClients.Pool.Close()

	ctx := context.Background()
	orgA := "org-user-test-a-" + uuid.New().String()[:6]
	orgB := "org-user-test-b-" + uuid.New().String()[:6]

	_, _ = dbClients.Pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, $2)", orgA, "Org A")
	_, _ = dbClients.Pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, $2)", orgB, "Org B")

	adminAEmail := fmt.Sprintf("admin-%s@test.io", uuid.New().String()[:6])
	adminAToken := registerAndGetToken(t, router, adminAEmail, orgA, models.RoleAdmin)

	viewerAEmail := fmt.Sprintf("viewer-%s@test.io", uuid.New().String()[:6])
	viewerAToken := registerAndGetToken(t, router, viewerAEmail, orgA, models.RoleViewer)

	adminBEmail := fmt.Sprintf("admin-%s@test.io", uuid.New().String()[:6])
	adminBToken := registerAndGetToken(t, router, adminBEmail, orgB, models.RoleAdmin)

	t.Run("RBAC - Viewer Forbidden from Listing Users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+viewerAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Admin - List Users in Organization Scoped", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data  []models.User `json:"data"`
			Total int           `json:"total"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.GreaterOrEqual(t, resp.Total, 2)
		for _, u := range resp.Data {
			assert.Equal(t, orgA, u.OrganizationID)
		}
	})

	var createdUserID string
	t.Run("Admin - Create New User with Valid Role", func(t *testing.T) {
		newUserEmail := fmt.Sprintf("operator-%s@test.io", uuid.New().String()[:6])
		body, _ := json.Marshal(map[string]string{
			"email":    newUserEmail,
			"password": "SecurePassword123!",
			"role":     models.RoleOperator,
		})

		req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data models.User `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, newUserEmail, resp.Data.Email)
		assert.Equal(t, models.RoleOperator, resp.Data.Role)
		assert.Equal(t, orgA, resp.Data.OrganizationID)
		createdUserID = resp.Data.ID
	})

	t.Run("Admin - Update User Role", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"role": models.RoleViewer,
		})

		path := fmt.Sprintf("/api/v1/users/%s/role", createdUserID)
		req, _ := http.NewRequest("PATCH", path, bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Cross-Org Isolation - Admin B Cannot Delete User in Org A", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/users/%s", createdUserID)
		req, _ := http.NewRequest("DELETE", path, nil)
		req.Header.Set("Authorization", "Bearer "+adminBToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Admin - Delete User Successfully", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/users/%s", createdUserID)
		req, _ := http.NewRequest("DELETE", path, nil)
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
