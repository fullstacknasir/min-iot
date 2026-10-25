package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

func TestDeviceAssignmentAndProfileLifecycle(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffix := uuid.New().String()[:6]
	adminEmail := fmt.Sprintf("admin_assign_%s@enterprise.io", suffix)
	userAEmail := fmt.Sprintf("user_a_%s@enterprise.io", suffix)
	userBEmail := fmt.Sprintf("user_b_%s@enterprise.io", suffix)
	password := "SecurePassword123!"

	// 1. Register Org & Admin
	regAdmin := models.RegisterRequest{
		OrganizationName: "Assignment Corp " + suffix,
		Email:            adminEmail,
		Password:         password,
	}
	resAdmin := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regAdmin, "")
	adminToken := resAdmin.AccessToken
	orgID := resAdmin.Organization.ID

	// 2. Register User A (viewer)
	regUserA := models.RegisterRequest{
		OrganizationID: orgID,
		Email:          userAEmail,
		Password:       password,
		Role:           models.RoleViewer,
	}
	resUserA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regUserA, "")
	userAToken := resUserA.AccessToken
	userAID := resUserA.User.ID

	// 3. Register User B (viewer)
	regUserB := models.RegisterRequest{
		OrganizationID: orgID,
		Email:          userBEmail,
		Password:       password,
		Role:           models.RoleViewer,
	}
	resUserB := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regUserB, "")
	userBToken := resUserB.AccessToken
	userBID := resUserB.User.ID

	// 4. Admin creates Device 1 assigned to User A
	device1ID := "dev-assign-1-" + suffix
	createReq1 := models.CreateDeviceRequest{
		ID:             device1ID,
		Name:           "Sensor For User A",
		DeviceType:     "temperature",
		AssignedUserID: &userAID,
	}
	wCreate1 := sendRequest(router, http.MethodPost, "/api/v1/devices", createReq1, adminToken)
	assert.Equal(t, http.StatusCreated, wCreate1.Code)

	// 5. Admin creates Device 2 assigned to User B
	device2ID := "dev-assign-2-" + suffix
	createReq2 := models.CreateDeviceRequest{
		ID:             device2ID,
		Name:           "Sensor For User B",
		DeviceType:     "humidity",
		AssignedUserID: &userBID,
	}
	wCreate2 := sendRequest(router, http.MethodPost, "/api/v1/devices", createReq2, adminToken)
	assert.Equal(t, http.StatusCreated, wCreate2.Code)

	// 6. User A lists devices -> must see Device 1, must NOT see Device 2
	wListA := sendRequest(router, http.MethodGet, "/api/v1/devices", nil, userAToken)
	assert.Equal(t, http.StatusOK, wListA.Code)
	assert.Contains(t, wListA.Body.String(), device1ID)
	assert.NotContains(t, wListA.Body.String(), device2ID)

	// 7. User B lists devices -> must see Device 2, must NOT see Device 1
	wListB := sendRequest(router, http.MethodGet, "/api/v1/devices", nil, userBToken)
	assert.Equal(t, http.StatusOK, wListB.Code)
	assert.Contains(t, wListB.Body.String(), device2ID)
	assert.NotContains(t, wListB.Body.String(), device1ID)

	// 8. Admin lists devices -> must see BOTH Device 1 and Device 2
	wListAdmin := sendRequest(router, http.MethodGet, "/api/v1/devices", nil, adminToken)
	assert.Equal(t, http.StatusOK, wListAdmin.Code)
	assert.Contains(t, wListAdmin.Body.String(), device1ID)
	assert.Contains(t, wListAdmin.Body.String(), device2ID)

	// 9. Profile tests: User A gets own profile
	wProfileGet := sendRequest(router, http.MethodGet, "/api/v1/auth/me", nil, userAToken)
	assert.Equal(t, http.StatusOK, wProfileGet.Code)
	assert.Contains(t, wProfileGet.Body.String(), userAEmail)

	// 10. User A updates profile (Name + New Password)
	newName := "User Alpha Updated"
	newPassword := "BrandNewPassword456!"
	updateProf := models.UpdateProfileRequest{
		Name:            &newName,
		CurrentPassword: &password,
		NewPassword:     &newPassword,
	}
	wProfilePut := sendRequest(router, http.MethodPut, "/api/v1/auth/me", updateProf, userAToken)
	assert.Equal(t, http.StatusOK, wProfilePut.Code)
	assert.Contains(t, wProfilePut.Body.String(), newName)

	// 11. Verify User A can now log in with the new password
	loginReq := models.LoginRequest{
		Email:    userAEmail,
		Password: newPassword,
	}
	wLoginNew := sendRequest(router, http.MethodPost, "/api/v1/auth/login", loginReq, "")
	assert.Equal(t, http.StatusOK, wLoginNew.Code)
	assert.Contains(t, wLoginNew.Body.String(), "access_token")
}
