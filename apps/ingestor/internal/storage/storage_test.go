package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miniot/apps/ingestor/internal/config"
	"miniot/apps/ingestor/internal/validator"
)

func setupTestStorage(t *testing.T) (*Storage, *config.Config) {
	cfg := config.LoadConfig()
	store, err := NewStorage(cfg)
	if err != nil {
		t.Skipf("Skipping storage tests (database unavailable): %v", err)
	}
	return store, cfg
}

func createTestDevice(t *testing.T, store *Storage, deviceID string) {
	ctx := context.Background()
	query := `INSERT INTO devices (id, name, device_type, status, organization_id)
	          VALUES ($1, $2, 'sensor', 'offline', 'org-default-001')
	          ON CONFLICT (id) DO NOTHING`
	_, err := store.Pool.Exec(ctx, query, deviceID, "Test Sensor "+deviceID)
	require.NoError(t, err)
}

func TestStorageSaveTelemetryDeviceAndServerTimestamps(t *testing.T) {
	store, _ := setupTestStorage(t)
	defer store.Pool.Close()

	ctx := context.Background()
	deviceID := "dev-ts-test-" + uuid.New().String()[:6]
	createTestDevice(t, store, deviceID)

	deviceTime := time.Date(2026, 10, 18, 14, 0, 0, 0, time.UTC)
	receivedAt := time.Now().UTC()

	data := &validator.ValidatedTelemetry{
		DeviceID:   deviceID,
		DeviceTime: deviceTime,
		ReceivedAt: receivedAt,
		Metrics: map[string]float64{
			"temperature": 25.4,
			"humidity":    61.2,
		},
	}

	err := store.SaveTelemetry(ctx, data)
	require.NoError(t, err)

	// Verify points in TimescaleDB hypertable
	points, err := store.GetTelemetryPoints(ctx, deviceID, "temperature", 10)
	require.NoError(t, err)
	require.Len(t, points, 1)

	assert.Equal(t, deviceTime.Unix(), points[0].Time.Unix())
	assert.WithinDuration(t, receivedAt, points[0].ReceivedAt, 2*time.Second)
	assert.Equal(t, 25.4, points[0].MetricValue)
	assert.Equal(t, "temperature", points[0].MetricName)
}

func TestStorageIdempotentDuplicates(t *testing.T) {
	store, _ := setupTestStorage(t)
	defer store.Pool.Close()

	ctx := context.Background()
	deviceID := "dev-idemp-test-" + uuid.New().String()[:6]
	createTestDevice(t, store, deviceID)

	deviceTime := time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)
	initialReceivedAt := time.Now().UTC().Add(-10 * time.Second)

	initialData := &validator.ValidatedTelemetry{
		DeviceID:   deviceID,
		DeviceTime: deviceTime,
		ReceivedAt: initialReceivedAt,
		Metrics: map[string]float64{
			"voltage": 220.0,
		},
	}

	// 1. Initial insert
	err := store.SaveTelemetry(ctx, initialData)
	require.NoError(t, err)

	// 2. Duplicate insert with exact same time and metric
	duplicateReceivedAt := time.Now().UTC()
	duplicateData := &validator.ValidatedTelemetry{
		DeviceID:   deviceID,
		DeviceTime: deviceTime,
		ReceivedAt: duplicateReceivedAt,
		Metrics: map[string]float64{
			"voltage": 220.5, // Updated metric reading on retry
		},
	}

	err = store.SaveTelemetry(ctx, duplicateData)
	require.NoError(t, err, "Duplicate insert must succeed idempotently without error")

	// Verify that exactly 1 record exists in hypertable for this device, time, and metric
	points, err := store.GetTelemetryPoints(ctx, deviceID, "voltage", 10)
	require.NoError(t, err)
	require.Len(t, points, 1, "There must be exactly one row, no duplicates")

	assert.Equal(t, 220.5, points[0].MetricValue)
	assert.Equal(t, deviceTime.Unix(), points[0].Time.Unix())
	assert.WithinDuration(t, initialReceivedAt, points[0].ReceivedAt, 2*time.Second)
}

func TestStorageOutOfOrderMessages(t *testing.T) {
	store, _ := setupTestStorage(t)
	defer store.Pool.Close()

	ctx := context.Background()
	deviceID := "dev-ooo-test-" + uuid.New().String()[:6]
	createTestDevice(t, store, deviceID)

	timeNewer := time.Date(2026, 10, 18, 16, 0, 0, 0, time.UTC)
	timeOlder := time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)

	// 1. Send newer message first (T2)
	newerData := &validator.ValidatedTelemetry{
		DeviceID:   deviceID,
		DeviceTime: timeNewer,
		ReceivedAt: time.Now().UTC(),
		Metrics: map[string]float64{
			"power": 150.0,
		},
	}
	err := store.SaveTelemetry(ctx, newerData)
	require.NoError(t, err)

	// Verify device last_seen is timeNewer
	lastSeenNewer, err := store.GetDeviceLastSeen(ctx, deviceID)
	require.NoError(t, err)
	require.NotNil(t, lastSeenNewer)
	assert.Equal(t, timeNewer.Unix(), lastSeenNewer.Unix())

	// Verify Redis cache has timeNewer
	cachedNewer, err := store.GetCachedLatestTelemetry(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, timeNewer.Unix(), cachedNewer.Timestamp.Unix())
	assert.Equal(t, 150.0, cachedNewer.Metrics["power"])

	// 2. Send older message later (T1, out-of-order)
	olderData := &validator.ValidatedTelemetry{
		DeviceID:   deviceID,
		DeviceTime: timeOlder,
		ReceivedAt: time.Now().UTC(),
		Metrics: map[string]float64{
			"power": 120.0,
		},
	}
	err = store.SaveTelemetry(ctx, olderData)
	require.NoError(t, err)

	// 3. Verify older data is stored in TimescaleDB hypertable
	points, err := store.GetTelemetryPoints(ctx, deviceID, "power", 10)
	require.NoError(t, err)
	assert.Len(t, points, 2, "Both out-of-order data points must be stored in TimescaleDB")

	// 4. Verify device last_seen did NOT regress to older timestamp
	lastSeenAfterOOO, err := store.GetDeviceLastSeen(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, timeNewer.Unix(), lastSeenAfterOOO.Unix(), "last_seen must NOT regress on out-of-order message")

	// 5. Verify Redis latest cache did NOT regress to older timestamp
	cachedAfterOOO, err := store.GetCachedLatestTelemetry(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, timeNewer.Unix(), cachedAfterOOO.Timestamp.Unix(), "Redis latest cache must retain newer timestamp")
	assert.Equal(t, 150.0, cachedAfterOOO.Metrics["power"])
}

func TestStorageRetentionPolicyExists(t *testing.T) {
	store, _ := setupTestStorage(t)
	defer store.Pool.Close()

	ctx := context.Background()
	exists, dropAfter, err := store.CheckRetentionPolicy(ctx, "telemetry")
	require.NoError(t, err)
	assert.True(t, exists, "30-day raw retention policy must exist")
	assert.Contains(t, dropAfter, "30 days")
}

func TestStorageContinuousAggregateExists(t *testing.T) {
	store, _ := setupTestStorage(t)
	defer store.Pool.Close()

	ctx := context.Background()
	exists, err := store.CheckContinuousAggregate(ctx, ContinuousAggViewName)
	require.NoError(t, err)
	assert.True(t, exists, "Continuous aggregate view telemetry_hourly must exist")
}
