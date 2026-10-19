package test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miniot/apps/ingestor/internal/config"
	"miniot/apps/ingestor/internal/mqtt"
	"miniot/apps/ingestor/internal/processor"
	"miniot/apps/ingestor/internal/storage"
)

const (
	TestDefaultTimeout = 5 * time.Second
)

func setupTestDeviceCredentials(t *testing.T, store *storage.Storage, deviceID, rawSecret string) {
	ctx := context.Background()

	_, _ = store.Pool.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ('org-default-001', 'Acme Corp') ON CONFLICT (id) DO NOTHING`)

	_, err := store.Pool.Exec(ctx, `INSERT INTO devices (id, name, device_type, status, organization_id) 
	                                VALUES ($1, $2, 'sensor', 'offline', 'org-default-001') 
	                                ON CONFLICT (id) DO NOTHING`, deviceID, "E2E Sensor "+deviceID)
	require.NoError(t, err)

	hasher := sha256.New()
	hasher.Write([]byte(rawSecret))
	secretHash := hex.EncodeToString(hasher.Sum(nil))

	credID := "cred-" + uuid.New().String()[:8]
	_, err = store.Pool.Exec(ctx, `INSERT INTO device_credentials (id, device_id, organization_id, credential_type, secret_hash, status) 
	                               VALUES ($1, $2, 'org-default-001', 'token', $3, 'active')`, credID, deviceID, secretHash)
	require.NoError(t, err)
}

func connectDeviceMQTT(brokerURI, clientID, password string) (paho.Client, error) {
	opts := paho.NewClientOptions().
		AddBroker(brokerURI).
		SetClientID(clientID).
		SetUsername(clientID).
		SetPassword(password).
		SetConnectTimeout(TestDefaultTimeout)

	client := paho.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return client, nil
}

func TestMQTTIngestorEndToEnd(t *testing.T) {
	cfg := config.LoadConfig()
	store, err := storage.NewStorage(cfg)
	if err != nil {
		t.Skipf("Skipping live MQTT test (storage unavailable): %v", err)
	}
	defer store.Pool.Close()

	proc, err := processor.NewTelemetryProcessor(store)
	require.NoError(t, err)

	cfg.MQTTClientID = "miniot-ingestor-test-" + uuid.New().String()[:6]
	subscriber := mqtt.NewSubscriber(cfg, proc)
	if err := subscriber.Start(); err != nil {
		t.Skipf("Skipping live MQTT test (EMQX broker unavailable or not running): %v", err)
	}
	defer subscriber.Stop()

	time.Sleep(500 * time.Millisecond)

	// 1. Provision a test device
	deviceID := "dev-e2e-" + uuid.New().String()[:6]
	deviceSecret := "dsec_test_" + uuid.New().String()[:8]
	setupTestDeviceCredentials(t, store, deviceID, deviceSecret)

	brokerURI := fmt.Sprintf("tcp://%s:%s", cfg.EMQXHost, cfg.EMQXPort)
	client, err := connectDeviceMQTT(brokerURI, deviceID, deviceSecret)
	if err != nil {
		t.Skipf("Skipping live device MQTT connect (API auth bridge unavailable): %v", err)
	}
	defer client.Disconnect(250)

	// 2. Publish valid telemetry message
	deviceTime := time.Date(2026, 10, 18, 19, 0, 0, 0, time.UTC)
	validPayload := fmt.Sprintf(`{
		"device_id": "%s",
		"timestamp": "%s",
		"metrics": {
			"temperature": 27.5,
			"humidity": 45.0
		}
	}`, deviceID, deviceTime.Format(time.RFC3339))

	telemetryTopic := fmt.Sprintf("devices/%s/telemetry", deviceID)
	pubToken := client.Publish(telemetryTopic, 1, false, []byte(validPayload))
	require.True(t, pubToken.WaitTimeout(TestDefaultTimeout), "Publish must complete within timeout")
	require.NoError(t, pubToken.Error())

	// 3. Await ingestion into TimescaleDB
	var points []storage.TelemetryPoint
	ctx := context.Background()
	for i := 0; i < 25; i++ {
		points, err = store.GetTelemetryPoints(ctx, deviceID, "temperature", 5)
		if err == nil && len(points) > 0 {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}

	require.NotEmpty(t, points, "Telemetry point must be persisted in TimescaleDB hypertable")
	assert.Equal(t, 27.5, points[0].MetricValue)
	assert.Equal(t, deviceTime.Unix(), points[0].Time.Unix(), "Device timestamp must match")
	assert.False(t, points[0].ReceivedAt.IsZero(), "Server receive timestamp must be populated")

	// 4. Verify Redis cache
	cached, err := store.GetCachedLatestTelemetry(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, 27.5, cached.Metrics["temperature"])
	assert.Equal(t, 45.0, cached.Metrics["humidity"])

	// 5. Publish malformed message - verify it doesn't corrupt hypertable
	malformedPayload := []byte(`{not valid json}`)
	malformedToken := client.Publish(telemetryTopic, 1, false, malformedPayload)
	require.True(t, malformedToken.WaitTimeout(TestDefaultTimeout))

	time.Sleep(300 * time.Millisecond)
	pointsAfter, err := store.GetTelemetryPoints(ctx, deviceID, "temperature", 5)
	require.NoError(t, err)
	assert.Len(t, pointsAfter, 1, "Malformed message must be rejected and not alter hypertable rows")
}
