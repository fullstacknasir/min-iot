package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIngestorHealthHandler(t *testing.T) {
	t.Run("Degraded Health When Storage and Subscriber are Nil", func(t *testing.T) {
		s := NewServer("8081", nil, nil)

		req, err := http.NewRequest("GET", "/health", nil)
		assert.NoError(t, err)

		w := httptest.NewRecorder()
		s.handleHealth(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		var resp HealthResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "degraded", resp.Status)
		assert.Equal(t, "miniot-ingestor", resp.Service)
		assert.Equal(t, "uninitialized", resp.Checks["timescaledb"])
		assert.Equal(t, "uninitialized", resp.Checks["redis"])
		assert.Equal(t, "disconnected", resp.Checks["mqtt_broker"])
	})
}
