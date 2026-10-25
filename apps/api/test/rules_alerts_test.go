package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

func TestRulesAlertsAndNotificationsLifecycle(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffix := uuid.New().String()[:6]
	adminEmailA := fmt.Sprintf("admin_ra_%s@enterprise.io", suffix)
	viewerEmailA := fmt.Sprintf("viewer_ra_%s@enterprise.io", suffix)
	adminEmailB := fmt.Sprintf("admin_rb_%s@enterprise.io", suffix)
	password := "SecurePassword123!"

	// 1. Register Admin in Org A
	regAdminA := models.RegisterRequest{
		OrganizationName: "Rules Org A " + suffix,
		Email:            adminEmailA,
		Password:         password,
	}
	resAdminA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regAdminA, "")
	adminTokenA := resAdminA.AccessToken
	orgIDA := resAdminA.Organization.ID

	// Register Viewer in Org A
	regViewerA := models.RegisterRequest{
		OrganizationID: orgIDA,
		Email:          viewerEmailA,
		Password:       password,
		Role:           models.RoleViewer,
	}
	resViewerA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regViewerA, "")
	viewerTokenA := resViewerA.AccessToken

	// Register Admin in Org B (for cross-org tenancy check)
	regAdminB := models.RegisterRequest{
		OrganizationName: "Rules Org B " + suffix,
		Email:            adminEmailB,
		Password:         password,
	}
	resAdminB := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regAdminB, "")
	adminTokenB := resAdminB.AccessToken

	// 2. Create Device in Org A
	deviceID := fmt.Sprintf("dev-ra-%s", suffix)
	devReq := models.CreateDeviceRequest{
		ID:         deviceID,
		Name:       "Thermal Reactor",
		DeviceType: "reactor",
	}
	wDev := sendRequest(router, http.MethodPost, "/api/v1/devices", devReq, adminTokenA)
	assert.Equal(t, http.StatusCreated, wDev.Code)

	// 3. Test RBAC: Viewer cannot create rule (Forbidden 403)
	rulePayload := models.CreateRuleRequest{
		Name:            "High Temp Alert",
		Description:     "Triggers when temperature exceeds 40C",
		DeviceID:        &deviceID,
		MetricName:      "temperature",
		Condition:       ">",
		Threshold:       40.0,
		Severity:        "critical",
		CooldownSeconds: ptrInt(10),
	}
	wRuleForbidden := sendRequest(router, http.MethodPost, "/api/v1/rules", rulePayload, viewerTokenA)
	assert.Equal(t, http.StatusForbidden, wRuleForbidden.Code)

	// 4. Admin creates rule (Created 201)
	wRuleCreated := sendRequest(router, http.MethodPost, "/api/v1/rules", rulePayload, adminTokenA)
	assert.Equal(t, http.StatusCreated, wRuleCreated.Code)

	var createdRuleRes struct {
		Data models.Rule `json:"data"`
	}
	err := json.Unmarshal(wRuleCreated.Body.Bytes(), &createdRuleRes)
	assert.NoError(t, err)
	ruleID := createdRuleRes.Data.ID
	assert.NotEmpty(t, ruleID)
	assert.Equal(t, "High Temp Alert", createdRuleRes.Data.Name)
	assert.Equal(t, 40.0, createdRuleRes.Data.Threshold)

	// 5. List rules (Viewer can view, Admin can view)
	wListRules := sendRequest(router, http.MethodGet, "/api/v1/rules", nil, viewerTokenA)
	assert.Equal(t, http.StatusOK, wListRules.Code)
	var listRulesRes struct {
		Data  []models.Rule `json:"data"`
		Total int           `json:"total"`
	}
	err = json.Unmarshal(wListRules.Body.Bytes(), &listRulesRes)
	assert.NoError(t, err)
	assert.Equal(t, 1, listRulesRes.Total)
	assert.Equal(t, ruleID, listRulesRes.Data[0].ID)

	// Cross-org check: Org B sees 0 rules
	wListRulesB := sendRequest(router, http.MethodGet, "/api/v1/rules", nil, adminTokenB)
	assert.Equal(t, http.StatusOK, wListRulesB.Code)
	var listRulesResB struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(wListRulesB.Body.Bytes(), &listRulesResB)
	assert.Equal(t, 0, listRulesResB.Total)

	// 6. Ingest Telemetry: temperature 25.0 (below 40.0) -> No alert
	ingestNormal := models.IngestTelemetryRequest{
		DeviceID: deviceID,
		Metrics:  map[string]float64{"temperature": 25.0},
	}
	wIngestNormal := sendRequest(router, http.MethodPost, "/api/v1/telemetry", ingestNormal, adminTokenA)
	assert.Equal(t, http.StatusAccepted, wIngestNormal.Code)

	wAlertsCheck0 := sendRequest(router, http.MethodGet, "/api/v1/alerts", nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wAlertsCheck0.Code)
	var alertsRes0 struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(wAlertsCheck0.Body.Bytes(), &alertsRes0)
	assert.Equal(t, 0, alertsRes0.Total, "Normal telemetry should not trigger any alert")

	// 7. Ingest Telemetry: temperature 48.5 (above 40.0) -> Trigger alert!
	ingestHigh := models.IngestTelemetryRequest{
		DeviceID: deviceID,
		Metrics:  map[string]float64{"temperature": 48.5},
	}
	wIngestHigh := sendRequest(router, http.MethodPost, "/api/v1/telemetry", ingestHigh, adminTokenA)
	assert.Equal(t, http.StatusAccepted, wIngestHigh.Code)

	// Verify alert was created
	wAlertsCheck1 := sendRequest(router, http.MethodGet, "/api/v1/alerts", nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wAlertsCheck1.Code)
	var alertsRes1 struct {
		Data  []models.Alert `json:"data"`
		Total int            `json:"total"`
	}
	err = json.Unmarshal(wAlertsCheck1.Body.Bytes(), &alertsRes1)
	assert.NoError(t, err)
	assert.Equal(t, 1, alertsRes1.Total, "High temperature must generate an alert")
	alertID := alertsRes1.Data[0].ID
	assert.Equal(t, "critical", alertsRes1.Data[0].Severity)
	assert.Equal(t, "active", alertsRes1.Data[0].Status)
	assert.Equal(t, 48.5, alertsRes1.Data[0].MetricValue)

	// Cooldown check: Immediate second high telemetry point within 10s cooldown should NOT create duplicate alert
	wIngestHigh2 := sendRequest(router, http.MethodPost, "/api/v1/telemetry", ingestHigh, adminTokenA)
	assert.Equal(t, http.StatusAccepted, wIngestHigh2.Code)
	wAlertsCheckCooldown := sendRequest(router, http.MethodGet, "/api/v1/alerts", nil, adminTokenA)
	var alertsResCooldown struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(wAlertsCheckCooldown.Body.Bytes(), &alertsResCooldown)
	assert.Equal(t, 1, alertsResCooldown.Total, "Cooldown must prevent alert spamming")

	// 8. Verify Notification was generated
	wNotifCount := sendRequest(router, http.MethodGet, "/api/v1/notifications/unread-count", nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wNotifCount.Code)
	var countRes struct {
		UnreadCount int `json:"unread_count"`
	}
	err = json.Unmarshal(wNotifCount.Body.Bytes(), &countRes)
	assert.NoError(t, err)
	assert.Equal(t, 1, countRes.UnreadCount)

	wNotifList := sendRequest(router, http.MethodGet, "/api/v1/notifications", nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wNotifList.Code)
	var notifListRes struct {
		Data []models.Notification `json:"data"`
	}
	_ = json.Unmarshal(wNotifList.Body.Bytes(), &notifListRes)
	assert.Equal(t, 1, len(notifListRes.Data))
	notifID := notifListRes.Data[0].ID

	// 9. Acknowledge Alert
	wAck := sendRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/alerts/%s/acknowledge", alertID), nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wAck.Code)

	wGetAlertAck := sendRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/alerts/%s", alertID), nil, adminTokenA)
	var getAlertRes struct {
		Data models.Alert `json:"data"`
	}
	_ = json.Unmarshal(wGetAlertAck.Body.Bytes(), &getAlertRes)
	assert.Equal(t, "acknowledged", getAlertRes.Data.Status)
	assert.NotNil(t, getAlertRes.Data.AcknowledgedAt)

	// 10. Resolve Alert
	wResolve := sendRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/alerts/%s/resolve", alertID), nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wResolve.Code)

	wGetAlertResolved := sendRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/alerts/%s", alertID), nil, adminTokenA)
	_ = json.Unmarshal(wGetAlertResolved.Body.Bytes(), &getAlertRes)
	assert.Equal(t, "resolved", getAlertRes.Data.Status)
	assert.NotNil(t, getAlertRes.Data.ResolvedAt)

	// 11. Mark Notification as read
	wMarkRead := sendRequest(router, http.MethodPatch, fmt.Sprintf("/api/v1/notifications/%s/read", notifID), nil, adminTokenA)
	assert.Equal(t, http.StatusOK, wMarkRead.Code)

	wNotifCountAfter := sendRequest(router, http.MethodGet, "/api/v1/notifications/unread-count", nil, adminTokenA)
	_ = json.Unmarshal(wNotifCountAfter.Body.Bytes(), &countRes)
	assert.Equal(t, 0, countRes.UnreadCount, "Unread count should be 0 after marking read")
}

func ptrInt(v int) *int {
	return &v
}
