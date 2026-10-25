package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/handlers"
	"miniot/apps/api/internal/models"
)

type PaginatedTestResponse[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

func TestAPIPaginationEndpoints(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffix := uuid.New().String()[:6]
	adminEmail := fmt.Sprintf("admin_page_%s@enterprise.io", suffix)
	password := "SecurePassword123!"

	// 1. Register Admin & Org
	regAdmin := models.RegisterRequest{
		OrganizationName: "Pagination Corp " + suffix,
		Email:            adminEmail,
		Password:         password,
	}
	resAdmin := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regAdmin, "")
	adminToken := resAdmin.AccessToken
	assert.NotEmpty(t, resAdmin.Organization.ID)

	// 2. Create 5 test devices
	for i := 1; i <= 5; i++ {
		devReq := models.CreateDeviceRequest{
			ID:         fmt.Sprintf("dev-page-%d-%s", i, suffix),
			Name:       fmt.Sprintf("Pagination Sensor %d", i),
			DeviceType: "sensor",
		}
		w := sendRequest(router, http.MethodPost, "/api/v1/devices", devReq, adminToken)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	// 3. Test Device Pagination: Page 1, Limit 2
	wDevPage1 := sendRequest(router, http.MethodGet, "/api/v1/devices?page=1&limit=2", nil, adminToken)
	assert.Equal(t, http.StatusOK, wDevPage1.Code)

	var devRes1 PaginatedTestResponse[models.Device]
	err := json.Unmarshal(wDevPage1.Body.Bytes(), &devRes1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(devRes1.Data))
	assert.Equal(t, 5, devRes1.Total)
	assert.Equal(t, 1, devRes1.Page)
	assert.Equal(t, 2, devRes1.Limit)
	assert.Equal(t, 3, devRes1.TotalPages)

	// 4. Test Device Pagination: Page 2, Limit 2
	wDevPage2 := sendRequest(router, http.MethodGet, "/api/v1/devices?page=2&limit=2", nil, adminToken)
	assert.Equal(t, http.StatusOK, wDevPage2.Code)

	var devRes2 PaginatedTestResponse[models.Device]
	_ = json.Unmarshal(wDevPage2.Body.Bytes(), &devRes2)
	assert.Equal(t, 2, len(devRes2.Data))
	assert.Equal(t, 2, devRes2.Page)
	assert.NotEqual(t, devRes1.Data[0].ID, devRes2.Data[0].ID)

	// 5. Test Device Pagination: Page 3, Limit 2 (last page, only 1 item)
	wDevPage3 := sendRequest(router, http.MethodGet, "/api/v1/devices?page=3&limit=2", nil, adminToken)
	assert.Equal(t, http.StatusOK, wDevPage3.Code)

	var devRes3 PaginatedTestResponse[models.Device]
	_ = json.Unmarshal(wDevPage3.Body.Bytes(), &devRes3)
	assert.Equal(t, 1, len(devRes3.Data))

	// 6. Invite 3 users to test user pagination
	for i := 1; i <= 3; i++ {
		userEmail := fmt.Sprintf("team_user_%d_%s@enterprise.io", i, suffix)
		reqUser := handlers.CreateOrgUserRequest{
			Email:    userEmail,
			Password: password,
			Role:     models.RoleViewer,
		}
		wUserCreate := sendRequest(router, http.MethodPost, "/api/v1/users", reqUser, adminToken)
		assert.Equal(t, http.StatusCreated, wUserCreate.Code)
	}

	// Total users in org is now 4 (1 admin + 3 viewers)
	wUserPage1 := sendRequest(router, http.MethodGet, "/api/v1/users?page=1&limit=2", nil, adminToken)
	assert.Equal(t, http.StatusOK, wUserPage1.Code)

	var userRes1 PaginatedTestResponse[models.User]
	_ = json.Unmarshal(wUserPage1.Body.Bytes(), &userRes1)
	assert.Equal(t, 2, len(userRes1.Data))
	assert.Equal(t, 4, userRes1.Total)
	assert.Equal(t, 1, userRes1.Page)
	assert.Equal(t, 2, userRes1.Limit)
	assert.Equal(t, 2, userRes1.TotalPages)
}
