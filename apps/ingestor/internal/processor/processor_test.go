package processor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miniot/apps/ingestor/internal/config"
	"miniot/apps/ingestor/internal/storage"
)

func setupTestProcessor(t *testing.T) (*TelemetryProcessor, *storage.Storage) {
	cfg := config.LoadConfig()
	store, err := storage.NewStorage(cfg)
	if err != nil {
		t.Skipf("Skipping processor tests (database unavailable): %v", err)
	}

	proc, err := NewTelemetryProcessor(store)
	require.NoError(t, err)

	return proc, store
}

func createTestDevice(t *testing.T, store *storage.Storage, deviceID string) {
	ctx := context.Background()
	query := `INSERT INTO devices (id, name, device_type, status, organization_id)
	          VALUES ($1, $2, 'sensor', 'offline', 'org-default-001')
	          ON CONFLICT (id) DO NOTHING`
	_, err := store.Pool.Exec(ctx, query, deviceID, "Processor Test "+deviceID)
	require.NoError(t, err)
}

func TestProcessorProcessValidTelemetryMessage(t *testing.T) {
	proc, store := setupTestProcessor(t)
	defer store.Pool.Close()

	ctx := context.Background()
	deviceID := "dev-proc-" + uuid.New().String()[:6]
	createTestDevice(t, store, deviceID)

	topic := fmt.Sprintf("devices/%s/telemetry", deviceID)
	payload := []byte(fmt.Sprintf(`{
		"device_id": "%s",
		"timestamp": "2026-10-18T17:00:00Z",
		"metrics": {
			"temperature": 26.8,
			"humidity": 48.5
		}
	}`, deviceID))

	err := proc.ProcessMessage(topic, payload)
	require.NoError(t, err)

	// Verify hypertable rows
	points, err := store.GetTelemetryPoints(ctx, deviceID, "temperature", 10)
	require.NoError(t, err)
	require.NotEmpty(t, points)
	assert.Equal(t, 26.8, points[0].MetricValue)
	assert.False(t, points[0].ReceivedAt.IsZero())
	assert.Equal(t, int64(1792342800), points[0].Time.Unix()) // 2026-10-18T17:00:00Z

	// Verify Redis cache
	cached, err := store.GetCachedLatestTelemetry(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, 26.8, cached.Metrics["temperature"])
	assert.Equal(t, 48.5, cached.Metrics["humidity"])
}

func TestProcessorRejectsMalformedAndInvalidMessages(t *testing.T) {
	proc, store := setupTestProcessor(t)
	defer store.Pool.Close()

	deviceID := "dev-reject-" + uuid.New().String()[:6]
	createTestDevice(t, store, deviceID)
	topic := fmt.Sprintf("devices/%s/telemetry", deviceID)

	invalidCases := []struct {
		name    string
		topic   string
		payload []byte
	}{
		{
			name:    "Malformed syntax",
			topic:   topic,
			payload: []byte(`{"not-closed-json`),
		},
		{
			name:    "Empty payload",
			topic:   topic,
			payload: []byte(``),
		},
		{
			name:    "Missing timestamp",
			topic:   topic,
			payload: []byte(`{"metrics": {"temperature": 20.0}}`),
		},
		{
			name:    "Missing metrics",
			topic:   topic,
			payload: []byte(`{"timestamp": "2026-10-18T17:00:00Z"}`),
		},
		{
			name:    "Empty metrics object",
			topic:   topic,
			payload: []byte(`{"timestamp": "2026-10-18T17:00:00Z", "metrics": {}}`),
		},
		{
			name:    "Non-numeric metric value",
			topic:   topic,
			payload: []byte(`{"timestamp": "2026-10-18T17:00:00Z", "metrics": {"temp": "hot"}}`),
		},
		{
			name:    "Device ID mismatch / spoofing",
			topic:   topic,
			payload: []byte(`{"device_id": "different-device", "timestamp": "2026-10-18T17:00:00Z", "metrics": {"temp": 20.0}}`),
		},
		{
			name:    "Invalid topic name",
			topic:   fmt.Sprintf("devices/%s/commands", deviceID),
			payload: []byte(`{"timestamp": "2026-10-18T17:00:00Z", "metrics": {"temp": 20.0}}`),
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			err := proc.ProcessMessage(tc.topic, tc.payload)
			assert.Error(t, err, "Expected processor to reject invalid message")
		})
	}
}

func TestProcessorDuplicateAndOutOfOrderHandling(t *testing.T) {
	proc, store := setupTestProcessor(t)
	defer store.Pool.Close()

	ctx := context.Background()
	deviceID := "dev-proc-ooo-" + uuid.New().String()[:6]
	createTestDevice(t, store, deviceID)
	topic := fmt.Sprintf("devices/%s/telemetry", deviceID)

	// Step 1: Send message with timestamp T2 (Newer: 18:00)
	t2 := "2026-10-18T18:00:00Z"
	payloadT2 := []byte(fmt.Sprintf(`{"timestamp": "%s", "metrics": {"temp": 30.0}}`, t2))
	err := proc.ProcessMessage(topic, payloadT2)
	require.NoError(t, err)

	lastSeenT2, err := store.GetDeviceLastSeen(ctx, deviceID)
	require.NoError(t, err)
	expectedT2, _ := time.Parse(time.RFC3339, t2)
	assert.Equal(t, expectedT2.Unix(), lastSeenT2.Unix())

	// Step 2: Send duplicate message with timestamp T2 (idempotent)
	err = proc.ProcessMessage(topic, payloadT2)
	require.NoError(t, err, "Duplicate message must be handled idempotently")

	pointsT2, err := store.GetTelemetryPoints(ctx, deviceID, "temp", 10)
	require.NoError(t, err)
	assert.Len(t, pointsT2, 1, "Duplicate must not create additional row")

	// Step 3: Send out-of-order message with timestamp T1 (Older: 17:00)
	t1 := "2026-10-18T17:00:00Z"
	payloadT1 := []byte(fmt.Sprintf(`{"timestamp": "%s", "metrics": {"temp": 28.0}}`, t1))
	err = proc.ProcessMessage(topic, payloadT1)
	require.NoError(t, err, "Out-of-order message must be stored successfully")

	// Both points exist in hypertable
	allPoints, err := store.GetTelemetryPoints(ctx, deviceID, "temp", 10)
	require.NoError(t, err)
	assert.Len(t, allPoints, 2, "Both T1 and T2 points must be stored")

	// Device last_seen remains at T2
	lastSeenAfterT1, err := store.GetDeviceLastSeen(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, expectedT2.Unix(), lastSeenAfterT1.Unix(), "last_seen must NOT be regressed by out-of-order T1")

	// Redis cache remains at T2
	cached, err := store.GetCachedLatestTelemetry(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, expectedT2.Unix(), cached.Timestamp.Unix(), "Redis latest cache must retain newest timestamp T2")
	assert.Equal(t, 30.0, cached.Metrics["temp"])
}
