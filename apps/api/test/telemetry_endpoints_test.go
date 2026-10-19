package test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miniot/apps/api/internal/database"
	"miniot/apps/api/internal/models"
)

func TestTelemetryOrganizationScopingAndEndpoints(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	ctx := context.Background()
	suffix := uuid.New().String()[:6]

	// ----------------------------------------------------
	// 1. Setup Two Independent Organizations
	// ----------------------------------------------------
	// Org A
	regOrgA := models.RegisterRequest{
		OrganizationName: "Alpha Tenancy Org " + suffix,
		Email:            fmt.Sprintf("admin_a_%s@tenancy.io", suffix),
		Password:         "SecurePassword123!",
	}
	resOrgA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regOrgA, "")
	tokenOrgA := resOrgA.AccessToken

	// Org B
	regOrgB := models.RegisterRequest{
		OrganizationName: "Beta Tenancy Org " + suffix,
		Email:            fmt.Sprintf("admin_b_%s@tenancy.io", suffix),
		Password:         "SecurePassword123!",
	}
	resOrgB := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", regOrgB, "")
	tokenOrgB := resOrgB.AccessToken

	// 2. Create Device A in Org A, Device B in Org B
	deviceAID := "dev-a-" + suffix
	deviceBID := "dev-b-" + suffix

	sendRequest(router, http.MethodPost, "/api/v1/devices", models.CreateDeviceRequest{
		ID:         deviceAID,
		Name:       "Alpha Sensor",
		DeviceType: "environmental_sensor",
	}, tokenOrgA)

	sendRequest(router, http.MethodPost, "/api/v1/devices", models.CreateDeviceRequest{
		ID:         deviceBID,
		Name:       "Beta Sensor",
		DeviceType: "environmental_sensor",
	}, tokenOrgB)

	// 3. Populate Telemetry for Device A and Device B
	timeT1 := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	timeT2 := time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)
	timeT3 := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

	insertTestTelemetry(t, dbClients, deviceAID, "temperature", 22.5, timeT1)
	insertTestTelemetry(t, dbClients, deviceAID, "temperature", 24.0, timeT2)
	insertTestTelemetry(t, dbClients, deviceAID, "temperature", 26.5, timeT3)
	insertTestTelemetry(t, dbClients, deviceAID, "humidity", 55.0, timeT3)

	insertTestTelemetry(t, dbClients, deviceBID, "temperature", 99.9, timeT3)
	insertTestTelemetry(t, dbClients, deviceBID, "humidity", 10.0, timeT3)

	// Update Redis latest cache for Device A and B
	cacheTestTelemetry(ctx, dbClients, deviceAID, map[string]float64{"temperature": 26.5, "humidity": 55.0}, timeT3)
	cacheTestTelemetry(ctx, dbClients, deviceBID, map[string]float64{"temperature": 99.9, "humidity": 10.0}, timeT3)

	// ----------------------------------------------------
	// 4. Test Latest Telemetry Endpoint
	// ----------------------------------------------------
	t.Run("Latest Telemetry - Authorized Org A reading Device A", func(t *testing.T) {
		w := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceAID+"/telemetry/latest", nil, tokenOrgA)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "26.5")
	})

	t.Run("Latest Telemetry - Cross-Org Isolation (Org A reading Device B)", func(t *testing.T) {
		w := sendRequest(router, http.MethodGet, "/api/v1/devices/"+deviceBID+"/telemetry/latest", nil, tokenOrgA)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "device not found in organization")
	})

	// ----------------------------------------------------
	// 5. Test History & Time-Range Queries
	// ----------------------------------------------------
	t.Run("History Query - Authorized Org A reading Device A with Time Range", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/devices/%s/telemetry/history?metric=temperature&from=%s&to=%s&order=ASC",
			deviceAID, timeT1.Format(time.RFC3339), timeT3.Format(time.RFC3339))
		w := sendRequest(router, http.MethodGet, url, nil, tokenOrgA)
		assert.Equal(t, http.StatusOK, w.Code)

		var res models.TelemetryHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Equal(t, 3, res.Count)
		assert.Equal(t, 22.5, res.Data[0].MetricValue)
		assert.Equal(t, 26.5, res.Data[2].MetricValue)
	})

	t.Run("History Query - Cross-Org Isolation (Org A reading Device B)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/devices/%s/telemetry/history?metric=temperature", deviceBID)
		w := sendRequest(router, http.MethodGet, url, nil, tokenOrgA)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "device not found in organization")
	})

	// ----------------------------------------------------
	// 6. Test Aggregations (time_bucket avg/min/max/count)
	// ----------------------------------------------------
	t.Run("Aggregates Query - Authorized Org A reading Device A", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/devices/%s/telemetry/aggregates?bucket=1 hour&metric=temperature", deviceAID)
		w := sendRequest(router, http.MethodGet, url, nil, tokenOrgA)
		assert.Equal(t, http.StatusOK, w.Code)

		var res models.TelemetryAggregatesResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.NotEmpty(t, res.Aggregates)
		assert.Equal(t, "1 hour", res.Bucket)
		assert.Equal(t, "temperature", res.Aggregates[0].MetricName)
	})

	t.Run("Aggregates Query - Cross-Org Isolation (Org A reading Device B)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/devices/%s/telemetry/aggregates?bucket=1 hour", deviceBID)
		w := sendRequest(router, http.MethodGet, url, nil, tokenOrgA)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "device not found in organization")
	})

	// ----------------------------------------------------
	// 7. Test Server-Sent Events (SSE) Live Stream
	// ----------------------------------------------------
	t.Run("SSE Live Stream - Cross-Org Isolation (Org A streaming Device B)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceBID+"/telemetry/live", nil)
		req.Header.Set("Authorization", "Bearer "+tokenOrgA)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "device not found in organization")
	})

	t.Run("SSE Live Stream - Authorized Streaming & Token Query Param", func(t *testing.T) {
		// Use ?token= query parameter to test browser EventSource compatibility
		reqCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		url := fmt.Sprintf("/api/v1/devices/%s/telemetry/live?token=%s", deviceAID, tokenOrgA)
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		// Concurrently publish a live telemetry message to Redis to trigger the SSE event
		go func() {
			time.Sleep(300 * time.Millisecond)
			livePayload := map[string]interface{}{
				"device_id": deviceAID,
				"metrics":   map[string]float64{"temperature": 28.5},
				"timestamp": time.Now().UTC(),
			}
			data, _ := json.Marshal(livePayload)
			_ = dbClients.Redis.Publish(ctx, "telemetry:device:"+deviceAID, data).Err()
		}()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")

		// Verify event stream output contains initial event or live telemetry
		bodyStr := w.Body.String()
		scanner := bufio.NewScanner(strings.NewReader(bodyStr))
		foundEvent := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:") {
				foundEvent = true
				break
			}
		}
		assert.True(t, foundEvent, "Expected SSE stream to emit events")
	})

	// ----------------------------------------------------
	// 8. Test OpenAPI Documentation Endpoints
	// ----------------------------------------------------
	t.Run("OpenAPI Specification and Swagger UI", func(t *testing.T) {
		// Spec YAML
		wYaml := sendRequest(router, http.MethodGet, "/docs/openapi.yaml", nil, "")
		assert.Equal(t, http.StatusOK, wYaml.Code)
		assert.Contains(t, wYaml.Body.String(), "openapi: 3.0.3")
		assert.Contains(t, wYaml.Body.String(), "/devices/{id}/telemetry/latest")
		assert.Contains(t, wYaml.Body.String(), "/devices/{id}/telemetry/aggregates")
		assert.Contains(t, wYaml.Body.String(), "/devices/{id}/telemetry/live")

		// Swagger UI HTML
		wDocs := sendRequest(router, http.MethodGet, "/docs", nil, "")
		assert.Equal(t, http.StatusOK, wDocs.Code)
		assert.Contains(t, wDocs.Body.String(), "SwaggerUIBundle")
	})
}

func insertTestTelemetry(t *testing.T, clients *database.Clients, deviceID, metric string, val float64, ts time.Time) {
	query := `INSERT INTO telemetry (time, received_at, device_id, metric_name, metric_value) 
	          VALUES ($1, $2, $3, $4, $5) 
	          ON CONFLICT (time, device_id, metric_name) DO UPDATE 
	          SET metric_value = EXCLUDED.metric_value`
	_, err := clients.Pool.Exec(context.Background(), query, ts, time.Now().UTC(), deviceID, metric, val)
	require.NoError(t, err)
}

func cacheTestTelemetry(ctx context.Context, clients *database.Clients, deviceID string, metrics map[string]float64, ts time.Time) {
	payload := map[string]interface{}{
		"device_id":  deviceID,
		"timestamp":  ts,
		"metrics":    metrics,
		"updated_at": time.Now().UTC(),
	}
	data, _ := json.Marshal(payload)
	_ = clients.Redis.Set(ctx, fmt.Sprintf("device:%s:latest", deviceID), data, 24*time.Hour).Err()
}
