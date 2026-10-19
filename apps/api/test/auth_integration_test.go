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

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestApp(t *testing.T) (*gin.Engine, *database.Clients, *config.Config) {
	cfg := config.LoadConfig()
	dbClients, err := database.InitDatabase(cfg)
	if err != nil {
		t.Skipf("Skipping database integration tests (database unavailable): %v", err)
	}

	ctx := context.Background()
	err = database.RunMigrations(ctx, dbClients.Pool)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(gin.Recovery())

	authRepo := repository.NewAuthRepository(dbClients.Pool)
	authHandler := handlers.NewAuthHandler(authRepo, cfg)
	rateLimiter := middleware.LoginRateLimiter(dbClients.Redis, cfg.LoginRateLimitAttempts, cfg.LoginRateLimitWindow)

	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", rateLimiter, authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
	}

	deviceRepo := repository.NewDeviceRepository(dbClients.Pool)
	telemetryRepo := repository.NewTelemetryRepository(dbClients.Pool, dbClients.Redis)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo)
	telemetryHandler := handlers.NewTelemetryHandler(telemetryRepo)
	emqxHandler := handlers.NewEMQXHandler(deviceRepo)

	emqxGroup := router.Group("/api/v1/emqx")
	{
		emqxGroup.POST("/auth", emqxHandler.HandleAuth)
		emqxGroup.POST("/acl", emqxHandler.HandleACL)
		emqxGroup.POST("/events", emqxHandler.HandleEvents)
		emqxGroup.POST("/webhook", emqxHandler.HandleEvents)
	}

	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// Device read endpoints
		protected.GET("/devices", deviceHandler.ListDevices)
		protected.GET("/devices/:id", deviceHandler.GetDevice)
		protected.GET("/devices/:id/credentials", deviceHandler.GetDeviceCredentials)

		// Device manage endpoints
		manage := protected.Group("/devices")
		manage.Use(middleware.RequireRole(models.RoleAdmin, models.RoleOperator))
		{
			manage.POST("", deviceHandler.CreateDevice)
			manage.PUT("/:id", deviceHandler.UpdateDevice)
			manage.DELETE("/:id", deviceHandler.DeleteDevice)
			manage.POST("/:id/credentials", deviceHandler.ProvisionDevice)
			manage.POST("/:id/enable", deviceHandler.EnableDevice)
			manage.POST("/:id/disable", deviceHandler.DisableDevice)
			manage.POST("/:id/revoke", deviceHandler.RevokeDevice)
		}

		// Telemetry endpoints
		protected.GET("/devices/:id/telemetry/latest", telemetryHandler.GetLatestTelemetry)
		protected.GET("/devices/:id/telemetry/history", telemetryHandler.GetHistory)
		protected.GET("/devices/:id/telemetry/aggregates", telemetryHandler.GetAggregates)
		protected.GET("/devices/:id/telemetry/live", telemetryHandler.StreamLiveTelemetry)
		protected.POST("/telemetry", middleware.RequireRole(models.RoleAdmin, models.RoleOperator), telemetryHandler.IngestTelemetry)
	}

	router.GET("/docs", handlers.ServeDocsUI)
	router.GET("/docs/openapi.yaml", handlers.ServeOpenAPISpec)

	return router, dbClients, cfg
}

func TestCompleteAuthenticationAndTenancyLifecycle(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	uniqueSuffix := uuid.New().String()[:6]
	emailOrgA := fmt.Sprintf("admin_a_%s@enterprise.io", uniqueSuffix)
	emailOrgB := fmt.Sprintf("admin_b_%s@enterprise.io", uniqueSuffix)
	password := "SecurePassword123!"

	// 1. Register Admin in Org A
	regPayloadA := models.RegisterRequest{
		OrganizationName: "Alpha Org " + uniqueSuffix,
		Email:            emailOrgA,
		Password:         password,
	}
	resA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regPayloadA, "")
	assert.NotEmpty(t, resA.AccessToken)
	assert.NotEmpty(t, resA.RefreshToken)
	assert.Equal(t, models.RoleAdmin, resA.User.Role)
	orgA := resA.Organization

	// 2. Register Admin in Org B
	regPayloadB := models.RegisterRequest{
		OrganizationName: "Beta Org " + uniqueSuffix,
		Email:            emailOrgB,
		Password:         password,
	}
	resB := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regPayloadB, "")
	assert.NotEmpty(t, resB.AccessToken)
	orgB := resB.Organization

	// 3. Duplicate email registration should fail
	wDup := sendRequest(router, http.MethodPost, "/api/v1/auth/register", regPayloadA, "")
	assert.Equal(t, http.StatusConflict, wDup.Code)

	// 4. Test Login
	loginPayload := models.LoginRequest{Email: emailOrgA, Password: password}
	loginRes := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/login", loginPayload, "")
	assert.NotEmpty(t, loginRes.AccessToken)
	assert.NotEmpty(t, loginRes.RefreshToken)

	// 5. Test Invalid Login Credentials
	wInvalidPass := sendRequest(router, http.MethodPost, "/api/v1/auth/login", models.LoginRequest{
		Email:    emailOrgA,
		Password: "WrongPassword!",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, wInvalidPass.Code)

	// 6. Test Refresh Token Rotation
	initialRefresh := loginRes.RefreshToken
	refreshRes := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/refresh", models.RefreshRequest{
		RefreshToken: initialRefresh,
	}, "")
	assert.NotEmpty(t, refreshRes.AccessToken)
	assert.NotEmpty(t, refreshRes.RefreshToken)
	assert.NotEqual(t, initialRefresh, refreshRes.RefreshToken, "Refresh token must rotate to a new value")

	// Attempting to reuse old refresh token must be rejected (Security: token rotation invalidation)
	wReused := sendRequest(router, http.MethodPost, "/api/v1/auth/refresh", models.RefreshRequest{
		RefreshToken: initialRefresh,
	}, "")
	assert.Equal(t, http.StatusUnauthorized, wReused.Code, "Reusing rotated token must fail")

	// 7. Test Logout
	wLogout := sendRequest(router, http.MethodPost, "/api/v1/auth/logout", models.LogoutRequest{
		RefreshToken: refreshRes.RefreshToken,
	}, "")
	assert.Equal(t, http.StatusOK, wLogout.Code)

	// Using logged-out refresh token should fail
	wAfterLogout := sendRequest(router, http.MethodPost, "/api/v1/auth/refresh", models.RefreshRequest{
		RefreshToken: refreshRes.RefreshToken,
	}, "")
	assert.Equal(t, http.StatusUnauthorized, wAfterLogout.Code)

	// 8. Test Multi-Tenancy Organization Scoping
	testMultiTenancyScoping(t, router, resA.AccessToken, resB.AccessToken, orgA.ID, orgB.ID)

	// 9. Test Role Based Access Control (RBAC)
	testRoleBasedAccessControl(t, router, orgA.ID, uniqueSuffix)
}

func testMultiTenancyScoping(t *testing.T, router *gin.Engine, tokenA, tokenB, orgAID, orgBID string) {
	// Org A creates device
	deviceA := models.CreateDeviceRequest{
		ID:         "dev-alpha-" + uuid.New().String()[:6],
		Name:       "Alpha Sensor",
		DeviceType: "temperature",
	}
	wCreateA := sendRequest(router, http.MethodPost, "/api/v1/devices", deviceA, tokenA)
	assert.Equal(t, http.StatusCreated, wCreateA.Code)

	// Org B creates device
	deviceB := models.CreateDeviceRequest{
		ID:         "dev-beta-" + uuid.New().String()[:6],
		Name:       "Beta Sensor",
		DeviceType: "pressure",
	}
	wCreateB := sendRequest(router, http.MethodPost, "/api/v1/devices", deviceB, tokenB)
	assert.Equal(t, http.StatusCreated, wCreateB.Code)

	// Org A lists devices: should see only Alpha, NOT Beta
	wListA := sendRequest(router, http.MethodGet, "/api/v1/devices", nil, tokenA)
	assert.Equal(t, http.StatusOK, wListA.Code)
	assert.Contains(t, wListA.Body.String(), deviceA.ID)
	assert.NotContains(t, wListA.Body.String(), deviceB.ID, "Org A must NOT see Org B's devices")

	// Org A queries Org B's device directly: must return 404 Not Found
	wDirectA := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceB.ID, nil, tokenA)
	assert.Equal(t, http.StatusNotFound, wDirectA.Code, "Direct access across orgs must return 404")

	// Org A queries telemetry for Org B's device: must return 404 Not Found
	wTelA := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceB.ID+"/telemetry/latest", nil, tokenA)
	assert.Equal(t, http.StatusNotFound, wTelA.Code, "Telemetry access across orgs must return 404")
}

func testRoleBasedAccessControl(t *testing.T, router *gin.Engine, orgID, suffix string) {
	// Register a Viewer in Org A
	viewerEmail := fmt.Sprintf("viewer_%s@enterprise.io", suffix)
	regViewer := models.RegisterRequest{
		OrganizationID: orgID,
		Email:          viewerEmail,
		Password:       "ViewerPass123!",
		Role:           models.RoleViewer,
	}
	resViewer := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regViewer, "")
	assert.Equal(t, models.RoleViewer, resViewer.User.Role)

	// Viewer can list devices
	wList := sendRequest(router, http.MethodGet, "/api/v1/devices", nil, resViewer.AccessToken)
	assert.Equal(t, http.StatusOK, wList.Code)

	// Viewer CANNOT create device (Forbidden 403)
	forbiddenCreate := models.CreateDeviceRequest{
		ID:         "dev-forbidden-" + suffix,
		Name:       "Illegal Device",
		DeviceType: "meter",
	}
	wPostForbidden := sendRequest(router, http.MethodPost, "/api/v1/devices", forbiddenCreate, resViewer.AccessToken)
	assert.Equal(t, http.StatusForbidden, wPostForbidden.Code, "Viewer role must be forbidden from creating devices")
}

func sendRequest(router *gin.Engine, method, url string, body interface{}, bearerToken string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func executeJSONRequest[T any](t *testing.T, router *gin.Engine, method, url string, body interface{}, token string) T {
	w := sendRequest(router, method, url, body, token)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected success status, got %d: %s", w.Code, w.Body.String())

	var result T
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	return result
}
