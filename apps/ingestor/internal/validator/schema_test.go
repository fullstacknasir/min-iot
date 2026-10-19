package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TestValidTopic      = "devices/dev-sensor-001/telemetry"
	TestMismatchedTopic = "devices/dev-sensor-999/telemetry"
	TestInvalidTopic    = "devices/dev-sensor-001/commands"
)

func TestValidatorValidStructuredPayload(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	now := time.Now().UTC()
	payload := []byte(`{
		"device_id": "dev-sensor-001",
		"timestamp": "2026-10-18T12:30:00Z",
		"metrics": {
			"temperature": 24.5,
			"humidity": 55.0
		}
	}`)

	result, err := val.ValidateAndParse(TestValidTopic, payload, now)
	require.NoError(t, err)
	assert.Equal(t, "dev-sensor-001", result.DeviceID)
	assert.Equal(t, 2026, result.DeviceTime.Year())
	assert.Equal(t, 24.5, result.Metrics["temperature"])
	assert.Equal(t, 55.0, result.Metrics["humidity"])
	assert.Equal(t, now, result.ReceivedAt)
}

func TestValidatorValidEpochTimestamps(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	now := time.Now().UTC()

	// Seconds
	payloadSec := []byte(`{
		"timestamp": 1729261800,
		"metrics": {"voltage": 230.5}
	}`)
	resultSec, err := val.ValidateAndParse(TestValidTopic, payloadSec, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1729261800), resultSec.DeviceTime.Unix())
	assert.Equal(t, 230.5, resultSec.Metrics["voltage"])

	// Milliseconds
	payloadMs := []byte(`{
		"timestamp": 1729261800000,
		"metrics": {"voltage": 230.5}
	}`)
	resultMs, err := val.ValidateAndParse(TestValidTopic, payloadMs, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1729261800), resultMs.DeviceTime.Unix())
}

func TestValidatorFlatPayload(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	now := time.Now().UTC()
	payload := []byte(`{
		"timestamp": "2026-10-18T10:00:00Z",
		"temperature": 21.3,
		"pressure": 1013.25
	}`)

	result, err := val.ValidateAndParse(TestValidTopic, payload, now)
	require.NoError(t, err)
	assert.Equal(t, 21.3, result.Metrics["temperature"])
	assert.Equal(t, 1013.25, result.Metrics["pressure"])
}

func TestValidatorRejectsMalformedJSON(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	malformedCases := [][]byte{
		[]byte(`{bad-json`),
		[]byte(``),
		[]byte(`12345`),
		[]byte(`["not", "an", "object"]`),
		[]byte(`{"timestamp": "2026-10-18T10:00:00Z"`),
	}

	for _, tc := range malformedCases {
		_, err := val.ValidateAndParse(TestValidTopic, tc, time.Now())
		assert.Error(t, err)
	}
}

func TestValidatorRejectsMissingOrInvalidTimestamp(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	testCases := [][]byte{
		[]byte(`{"metrics": {"temperature": 20.0}}`),
		[]byte(`{"timestamp": "", "metrics": {"temperature": 20.0}}`),
		[]byte(`{"timestamp": "invalid-date", "metrics": {"temperature": 20.0}}`),
		[]byte(`{"timestamp": -100, "metrics": {"temperature": 20.0}}`),
	}

	for _, tc := range testCases {
		_, err := val.ValidateAndParse(TestValidTopic, tc, time.Now())
		assert.Error(t, err)
	}
}

func TestValidatorRejectsEmptyOrInvalidMetrics(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	testCases := [][]byte{
		[]byte(`{"timestamp": "2026-10-18T10:00:00Z", "metrics": {}}`),
		[]byte(`{"timestamp": "2026-10-18T10:00:00Z", "metrics": {"temp": "warm"}}`),
		[]byte(`{"timestamp": "2026-10-18T10:00:00Z", "metrics": {"temp": true}}`),
		[]byte(`{"timestamp": "2026-10-18T10:00:00Z"}`),
	}

	for _, tc := range testCases {
		_, err := val.ValidateAndParse(TestValidTopic, tc, time.Now())
		assert.Error(t, err)
	}
}

func TestValidatorRejectsDeviceIDSpoofing(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	payload := []byte(`{
		"device_id": "dev-sensor-001",
		"timestamp": "2026-10-18T10:00:00Z",
		"metrics": {"temperature": 25.0}
	}`)

	// Topic device_id is dev-sensor-999, but payload device_id is dev-sensor-001
	_, err = val.ValidateAndParse(TestMismatchedTopic, payload, time.Now())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDeviceIDMismatch)
}

func TestValidatorRejectsInvalidTopicFormat(t *testing.T) {
	val, err := NewTelemetryValidator()
	require.NoError(t, err)

	payload := []byte(`{
		"timestamp": "2026-10-18T10:00:00Z",
		"metrics": {"temperature": 25.0}
	}`)

	invalidTopics := []string{
		"devices/dev-001/commands",
		"devices/dev-001",
		"devices//telemetry",
		"telemetry",
	}

	for _, top := range invalidTopics {
		_, err := val.ValidateAndParse(top, payload, time.Now())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTopicFormat)
	}
}
