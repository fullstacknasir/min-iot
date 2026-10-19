package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

func TestEMQXAuthenticationAndAuthorization(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffix := uuid.New().String()[:6]
	adminEmail := fmt.Sprintf("admin_emqx_%s@enterprise.io", suffix)
	password := "SecurePassword123!"

	// 1. Setup Organization and Devices
	regAdmin := models.RegisterRequest{
		OrganizationName: "MQTT Corp " + suffix,
		Email:            adminEmail,
		Password:         password,
	}
	resAdmin := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regAdmin, "")
	adminToken := resAdmin.AccessToken

	deviceAID := "dev-a-" + suffix
	deviceBID := "dev-b-" + suffix

	// Create Device A and Device B
	sendRequest(router, http.MethodPost, "/api/v1/devices", models.CreateDeviceRequest{ID: deviceAID, Name: "Device Alpha", DeviceType: "sensor"}, adminToken)
	sendRequest(router, http.MethodPost, "/api/v1/devices", models.CreateDeviceRequest{ID: deviceBID, Name: "Device Beta", DeviceType: "sensor"}, adminToken)

	// Provision credentials for Device A and Device B
	provA := executeJSONRequest[models.ProvisionDeviceResponse](t, router, http.MethodPost, "/api/v1/devices/"+deviceAID+"/credentials", nil, adminToken)
	provB := executeJSONRequest[models.ProvisionDeviceResponse](t, router, http.MethodPost, "/api/v1/devices/"+deviceBID+"/credentials", nil, adminToken)
	assert.NotEmpty(t, provA.Token)
	assert.NotEmpty(t, provB.Token)

	// ----------------------------------------------------
	// 2. Test EMQX Authentication Hook (/api/v1/emqx/auth)
	// ----------------------------------------------------

	// Valid credentials -> Allow
	authValid := executeJSONRequest[models.EMQXAuthResponse](t, router, http.MethodPost, "/api/v1/emqx/auth", models.EMQXAuthRequest{
		ClientID: deviceAID,
		Username: deviceAID,
		Password: provA.Token,
	}, "")
	assert.Equal(t, models.EMQXResultAllow, authValid.Result)

	// Invalid password -> Deny
	authWrongPass := executeJSONRequest[models.EMQXAuthResponse](t, router, http.MethodPost, "/api/v1/emqx/auth", models.EMQXAuthRequest{
		ClientID: deviceAID,
		Username: deviceAID,
		Password: "WrongPassword!",
	}, "")
	assert.Equal(t, models.EMQXResultDeny, authWrongPass.Result)

	// Unknown device ID -> Deny
	authUnknown := executeJSONRequest[models.EMQXAuthResponse](t, router, http.MethodPost, "/api/v1/emqx/auth", models.EMQXAuthRequest{
		ClientID: "dev-non-existent",
		Username: "dev-non-existent",
		Password: provA.Token,
	}, "")
	assert.Equal(t, models.EMQXResultDeny, authUnknown.Result)

	// Disabled device -> Deny
	sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceAID+"/disable", nil, adminToken)
	authDisabled := executeJSONRequest[models.EMQXAuthResponse](t, router, http.MethodPost, "/api/v1/emqx/auth", models.EMQXAuthRequest{
		ClientID: deviceAID,
		Username: deviceAID,
		Password: provA.Token,
	}, "")
	assert.Equal(t, models.EMQXResultDeny, authDisabled.Result, "Disabled device must be denied authentication")

	// Re-enable device
	sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceAID+"/enable", nil, adminToken)

	// Revoked device -> Deny
	sendRequest(router, http.MethodPost, "/api/v1/devices/"+deviceAID+"/revoke", nil, adminToken)
	authRevoked := executeJSONRequest[models.EMQXAuthResponse](t, router, http.MethodPost, "/api/v1/emqx/auth", models.EMQXAuthRequest{
		ClientID: deviceAID,
		Username: deviceAID,
		Password: provA.Token,
	}, "")
	assert.Equal(t, models.EMQXResultDeny, authRevoked.Result, "Revoked device must be denied authentication")

	// ----------------------------------------------------
	// 3. Test EMQX Authorization Hook (/api/v1/emqx/acl)
	// ----------------------------------------------------

	// Device B: Publish to its OWN telemetry topic -> ALLOW
	ownTelemetryTopic := fmt.Sprintf("devices/%s/telemetry", deviceBID)
	aclOwnPub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionPublish,
		Topic:    ownTelemetryTopic,
	}, "")
	assert.Equal(t, models.EMQXResultAllow, aclOwnPub.Result, "Device must be allowed to publish to its own telemetry topic")

	// Device B: Attempt to publish to ANOTHER device's topic -> DENY
	otherTelemetryTopic := fmt.Sprintf("devices/%s/telemetry", deviceAID)
	aclOtherPub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionPublish,
		Topic:    otherTelemetryTopic,
	}, "")
	assert.Equal(t, models.EMQXResultDeny, aclOtherPub.Result, "Device must NOT be allowed to publish to another device's topic")

	// Device B: Attempt to publish to arbitrary topic -> DENY
	aclArbitraryPub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionPublish,
		Topic:    "system/broadcast",
	}, "")
	assert.Equal(t, models.EMQXResultDeny, aclArbitraryPub.Result, "Device must NOT be allowed to publish to arbitrary topics")

	// Device B: Subscribe to its OWN command topic -> ALLOW
	ownCmdTopic := fmt.Sprintf("devices/%s/commands", deviceBID)
	aclOwnSub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionSubscribe,
		Topic:    ownCmdTopic,
	}, "")
	assert.Equal(t, models.EMQXResultAllow, aclOwnSub.Result, "Device must be allowed to subscribe to its own command topic")

	// Device B: Subscribe to wildcard sub-topics of its command topic -> ALLOW
	ownWildcardSub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionSubscribe,
		Topic:    fmt.Sprintf("devices/%s/commands/#", deviceBID),
	}, "")
	assert.Equal(t, models.EMQXResultAllow, ownWildcardSub.Result)

	// Device B: Attempt to subscribe to ANOTHER device's command topic -> DENY
	otherCmdTopic := fmt.Sprintf("devices/%s/commands", deviceAID)
	aclOtherSub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionSubscribe,
		Topic:    otherCmdTopic,
	}, "")
	assert.Equal(t, models.EMQXResultDeny, aclOtherSub.Result, "Device must NOT be allowed to subscribe to another device's commands")

	// Device B: Attempt to subscribe to global wildcard '#' -> DENY
	aclGlobalSub := executeJSONRequest[models.EMQXACLResponse](t, router, http.MethodPost, "/api/v1/emqx/acl", models.EMQXACLRequest{
		ClientID: deviceBID,
		Username: deviceBID,
		Action:   models.EMQXActionSubscribe,
		Topic:    "#",
	}, "")
	assert.Equal(t, models.EMQXResultDeny, aclGlobalSub.Result, "Device must NOT be allowed to subscribe to global wildcard")

	// ----------------------------------------------------
	// 4. Test Connect / Disconnect Status Tracking & last_seen
	// ----------------------------------------------------

	// Connect event
	now := time.Now().Unix()
	wConnect := sendRequest(router, http.MethodPost, "/api/v1/emqx/events", models.EMQXEventPayload{
		Event:       "client_connected",
		ClientID:    deviceBID,
		ConnectedAt: &now,
	}, "")
	assert.Equal(t, http.StatusOK, wConnect.Code)

	// Verify device status is online and last_seen is updated
	deviceAfterConnect := executeJSONRequest[map[string]interface{}](t, router, http.MethodGet, "/api/v1/devices/"+deviceBID, nil, adminToken)
	deviceDataConn := deviceAfterConnect["data"].(map[string]interface{})
	assert.Equal(t, models.DeviceStatusOnline, deviceDataConn["status"])
	assert.NotNil(t, deviceDataConn["last_seen"])

	// Disconnect event
	wDisconnect := sendRequest(router, http.MethodPost, "/api/v1/emqx/events", models.EMQXEventPayload{
		Event:          "client_disconnected",
		ClientID:       deviceBID,
		DisconnectedAt: &now,
		Reason:         "tcp_closed",
	}, "")
	assert.Equal(t, http.StatusOK, wDisconnect.Code)

	// Verify device status is offline and last_seen is preserved
	deviceAfterDisconnect := executeJSONRequest[map[string]interface{}](t, router, http.MethodGet, "/api/v1/devices/"+deviceBID, nil, adminToken)
	deviceDataDisconn := deviceAfterDisconnect["data"].(map[string]interface{})
	assert.Equal(t, models.DeviceStatusOffline, deviceDataDisconn["status"])
	assert.NotNil(t, deviceDataDisconn["last_seen"])
}
