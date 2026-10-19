package test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

func TestDeviceManagementLifecycle(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffix := uuid.New().String()[:6]
	adminEmail := fmt.Sprintf("admin_dev_%s@enterprise.io", suffix)
	operatorEmail := fmt.Sprintf("operator_dev_%s@enterprise.io", suffix)
	viewerEmail := fmt.Sprintf("viewer_dev_%s@enterprise.io", suffix)
	password := "SecurePassword123!"

	// Register Admin & Organization
	regAdmin := models.RegisterRequest{
		OrganizationName: "Device Corp " + suffix,
		Email:            adminEmail,
		Password:         password,
	}
	resAdmin := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regAdmin, "")
	adminToken := resAdmin.AccessToken
	orgID := resAdmin.Organization.ID

	// Register Operator in same Org
	regOp := models.RegisterRequest{
		OrganizationID: orgID,
		Email:          operatorEmail,
		Password:       password,
		Role:           models.RoleOperator,
	}
	resOp := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regOp, "")
	operatorToken := resOp.AccessToken

	// Register Viewer in same Org
	regView := models.RegisterRequest{
		OrganizationID: orgID,
		Email:          viewerEmail,
		Password:       password,
		Role:           models.RoleViewer,
	}
	resView := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regView, "")
	viewerToken := resView.AccessToken

	// Register another organization to test multi-tenancy isolation
	otherOrgAdminEmail := fmt.Sprintf("other_org_%s@enterprise.io", suffix)
	regOtherOrg := models.RegisterRequest{
		OrganizationName: "Other Corp " + suffix,
		Email:            otherOrgAdminEmail,
		Password:         password,
	}
	resOtherOrg := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regOtherOrg, "")
	otherOrgToken := resOtherOrg.AccessToken

	// 1. CREATE Device (by Operator)
	deviceID := "dev-test-" + suffix
	createReq := models.CreateDeviceRequest{
		ID:         deviceID,
		Name:       "Initial Sensor",
		DeviceType: "temperature_sensor",
		Metadata: map[string]interface{}{
			"location": "Warehouse 4",
			"tier":     1,
		},
	}
	wCreate := sendRequest(router, http.MethodPost, "/api/v1/devices", createReq, operatorToken)
	assert.Equal(t, http.StatusCreated, wCreate.Code)

	// 2. READ (List) Device (by Viewer)
	wList := sendRequest(router, http.MethodGet, "/api/v1/devices", nil, viewerToken)
	assert.Equal(t, http.StatusOK, wList.Code)
	assert.Contains(t, wList.Body.String(), deviceID)

	// 3. READ (Single) Device (by Viewer)
	wGet := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceID, nil, viewerToken)
	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Contains(t, wGet.Body.String(), "Initial Sensor")

	// 4. UPDATE Device (by Operator)
	updatedName := "Calibrated Sensor Alpha"
	updatedType := "temp_hum_combo"
	updateReq := models.UpdateDeviceRequest{
		Name:       &updatedName,
		DeviceType: &updatedType,
		Metadata: map[string]interface{}{
			"location":   "Warehouse 4",
			"calibrated": true,
		},
	}
	wUpdate := sendRequest(router, http.MethodPut, "/api/v1/devices/"+deviceID, updateReq, operatorToken)
	assert.Equal(t, http.StatusOK, wUpdate.Code)
	assert.Contains(t, wUpdate.Body.String(), updatedName)
	assert.Contains(t, wUpdate.Body.String(), updatedType)

	// 5. PROVISION per-device credentials (shown once, stored hashed)
	provRes := executeJSONRequest[models.ProvisionDeviceResponse](t, router, http.MethodPost, "/api/v1/devices/"+deviceID+"/credentials", nil, adminToken)
	assert.Equal(t, deviceID, provRes.DeviceID)
	assert.NotEmpty(t, provRes.Token, "Raw token must be returned upon provisioning")
	assert.True(t, strings.HasPrefix(provRes.Token, "dsec_"), "Token must have dsec_ prefix")
	assert.Equal(t, models.CredentialStatusActive, provRes.Status)

	// Verify reading credentials does NOT expose raw secret
	credGetRes := executeJSONRequest[map[string]interface{}](t, router, http.MethodGet, "/api/v1/devices/"+deviceID+"/credentials", nil, viewerToken)
	credData := credGetRes["data"].(map[string]interface{})
	assert.Equal(t, models.CredentialStatusActive, credData["status"])
	assert.Nil(t, credData["secret_hash"], "Raw secret or hash must never be returned")
	assert.Nil(t, credData["token"], "Raw token must never be returned on read")

	// 6. DISABLE device (by Operator)
	wDisable := sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/disable", nil, operatorToken)
	assert.Equal(t, http.StatusOK, wDisable.Code)
	assert.Contains(t, wDisable.Body.String(), models.DeviceStatusDisabled)

	// Verify device status is disabled
	wCheckDisabled := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceID, nil, viewerToken)
	assert.Contains(t, wCheckDisabled.Body.String(), models.DeviceStatusDisabled)

	// 7. ENABLE device (by Admin)
	wEnable := sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/enable", nil, adminToken)
	assert.Equal(t, http.StatusOK, wEnable.Code)
	assert.Contains(t, wEnable.Body.String(), models.DeviceStatusOffline)

	// 8. REVOKE device (by Admin)
	wRevoke := sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/revoke", nil, adminToken)
	assert.Equal(t, http.StatusOK, wRevoke.Code)
	assert.Contains(t, wRevoke.Body.String(), models.DeviceStatusRevoked)

	// Verify credentials status is revoked
	wCheckCredRevoked := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceID+"/credentials", nil, viewerToken)
	assert.Contains(t, wCheckCredRevoked.Body.String(), models.CredentialStatusRevoked)

	// 9. RBAC: Viewer is READ-ONLY and cannot manage
	testViewerPermissions(t, router, deviceID, viewerToken)

	// 10. MULTI-TENANCY ISOLATION: Other organization cannot manage or access device
	testMultiTenancyDeviceIsolation(t, router, deviceID, otherOrgToken)

	// 11. DELETE Device (by Admin)
	wDelete := sendRequest(router, http.MethodDelete, "/api/v1/devices/"+deviceID, nil, adminToken)
	assert.Equal(t, http.StatusOK, wDelete.Code)

	// Subsequent GET must return 404
	wGetDeleted := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceID, nil, adminToken)
	assert.Equal(t, http.StatusNotFound, wGetDeleted.Code)
}

func testViewerPermissions(t *testing.T, router *gin.Engine, deviceID, viewerToken string) {
	// Viewer cannot create device
	wCreate := sendRequest(router, http.MethodPost, "/api/v1/devices", models.CreateDeviceRequest{Name: "X", DeviceType: "Y"}, viewerToken)
	assert.Equal(t, http.StatusForbidden, wCreate.Code)

	// Viewer cannot update device
	newName := "Illegal"
	wUpdate := sendRequest(router, http.MethodPut, "/api/v1/devices/"+deviceID, models.UpdateDeviceRequest{Name: &newName}, viewerToken)
	assert.Equal(t, http.StatusForbidden, wUpdate.Code)

	// Viewer cannot provision credentials
	wProv := sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/credentials", nil, viewerToken)
	assert.Equal(t, http.StatusForbidden, wProv.Code)

	// Viewer cannot enable, disable, or revoke device
	assert.Equal(t, http.StatusForbidden, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/enable", nil, viewerToken).Code)
	assert.Equal(t, http.StatusForbidden, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/disable", nil, viewerToken).Code)
	assert.Equal(t, http.StatusForbidden, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/revoke", nil, viewerToken).Code)

	// Viewer cannot delete device
	wDelete := sendRequest(router, http.MethodDelete, "/api/v1/devices/"+deviceID, nil, viewerToken)
	assert.Equal(t, http.StatusForbidden, wDelete.Code)
}

func testMultiTenancyDeviceIsolation(t *testing.T, router *gin.Engine, deviceID, otherOrgToken string) {
	// Cross-org read
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceID, nil, otherOrgToken).Code)

	// Cross-org update
	newName := "Hacked"
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodPut, "/api/v1/devices/"+deviceID, models.UpdateDeviceRequest{Name: &newName}, otherOrgToken).Code)

	// Cross-org provision
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/credentials", nil, otherOrgToken).Code)

	// Cross-org enable / disable / revoke
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/enable", nil, otherOrgToken).Code)
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/disable", nil, otherOrgToken).Code)
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceID+"/revoke", nil, otherOrgToken).Code)

	// Cross-org delete
	assert.Equal(t, http.StatusNotFound, sendRequest(router, http.MethodDelete, "/api/v1/devices/"+deviceID, nil, otherOrgToken).Code)
}
