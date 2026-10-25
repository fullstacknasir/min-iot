package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
	"miniot/apps/api/internal/rules"
)

const (
	SSEKeepaliveInterval = 15 * time.Second
	DefaultHistoryLimit  = 100
	DefaultBucketSize    = "1 hour"
)

type TelemetryHandler struct {
	repo       *repository.TelemetryRepository
	ruleEngine *rules.RuleEngine
}

func NewTelemetryHandler(repo *repository.TelemetryRepository) *TelemetryHandler {
	return &TelemetryHandler{repo: repo}
}

func (h *TelemetryHandler) WithRuleEngine(engine *rules.RuleEngine) *TelemetryHandler {
	h.ruleEngine = engine
	return h
}

// GetLatestTelemetry returns the real-time cached telemetry for a device scoped to caller's org.
func (h *TelemetryHandler) GetLatestTelemetry(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	data, err := h.repo.GetLatestTelemetry(c.Request.Context(), deviceID, orgID)
	if err != nil {
		handleRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_id": deviceID, "data": data})
}

// GetHistory queries TimescaleDB hypertable for time-range telemetry points scoped to caller's org.
func (h *TelemetryHandler) GetHistory(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	fromTime, toTime, err := parseTimeRangeQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	metric := c.Query("metric")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	order := c.DefaultQuery("order", "DESC")

	history, err := h.repo.GetTimeRangeHistory(c.Request.Context(), deviceID, orgID, metric, fromTime, toTime, limit, order)
	if err != nil {
		handleRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.TelemetryHistoryResponse{
		DeviceID: deviceID,
		Count:    len(history),
		Data:     history,
	})
}

// GetAggregates calculates downsampled telemetry averages, minimums, maximums, and counts across time buckets.
func (h *TelemetryHandler) GetAggregates(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	fromTime, toTime, err := parseTimeRangeQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	bucket := c.DefaultQuery("bucket", DefaultBucketSize)
	metric := c.Query("metric")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	aggs, err := h.repo.GetAggregates(c.Request.Context(), deviceID, orgID, bucket, metric, fromTime, toTime, limit)
	if err != nil {
		handleRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.TelemetryAggregatesResponse{
		DeviceID:   deviceID,
		Bucket:     bucket,
		Count:      len(aggs),
		Aggregates: aggs,
	})
}

// StreamLiveTelemetry opens a Server-Sent Events (SSE) stream for live device telemetry events.
func (h *TelemetryHandler) StreamLiveTelemetry(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	if err := h.repo.VerifyDeviceBelongsToOrg(c.Request.Context(), deviceID, orgID); err != nil {
		handleRepositoryError(c, err)
		return
	}

	setSSEHeaders(c)
	h.sendInitialSSEData(c, deviceID, orgID)
	h.streamLiveRedisUpdates(c, deviceID)
}

func (h *TelemetryHandler) streamLiveRedisUpdates(c *gin.Context, deviceID string) {
	pubsub := h.repo.SubscribeLiveTelemetry(c.Request.Context(), deviceID)
	defer pubsub.Close()

	ch := pubsub.Channel()
	ticker := time.NewTicker(SSEKeepaliveInterval)
	defer ticker.Stop()

	w := c.Writer
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: telemetry\ndata: %s\n\n", msg.Payload)
			w.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			w.Flush()
		}
	}
}

func (h *TelemetryHandler) sendInitialSSEData(c *gin.Context, deviceID, orgID string) {
	latestData, err := h.repo.GetLatestTelemetry(c.Request.Context(), deviceID, orgID)
	if err == nil && latestData != nil {
		if rawJSON, err := json.Marshal(latestData); err == nil {
			fmt.Fprintf(c.Writer, "event: initial\ndata: %s\n\n", string(rawJSON))
			c.Writer.Flush()
		}
	}
}

// IngestTelemetry handles direct HTTP telemetry submissions scoped to organization.
func (h *TelemetryHandler) IngestTelemetry(c *gin.Context) {
	var req models.IngestTelemetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	timestamp := time.Now().UTC()
	if req.Timestamp != nil {
		timestamp = req.Timestamp.UTC()
	}

	if err := h.repo.SaveBatch(c.Request.Context(), req.DeviceID, orgID, timestamp, req.Metrics); err != nil {
		handleRepositoryError(c, err)
		return
	}

	if h.ruleEngine != nil {
		_ = h.ruleEngine.EvaluateBatch(c.Request.Context(), orgID, req.DeviceID, req.Metrics, timestamp)
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "ingested", "device_id": req.DeviceID})
}

func handleRepositoryError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrDeviceNotFoundInOrg) {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func parseTimeRangeQuery(c *gin.Context) (*time.Time, *time.Time, error) {
	var fromTime, toTime *time.Time

	if fromStr := c.Query("from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid 'from' timestamp; must be RFC3339: %w", err)
		}
		fromTime = &parsed
	}

	if toStr := c.Query("to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid 'to' timestamp; must be RFC3339: %w", err)
		}
		toTime = &parsed
	}

	if fromTime != nil && toTime != nil && fromTime.After(*toTime) {
		return nil, nil, errors.New("'from' timestamp cannot be after 'to' timestamp")
	}

	return fromTime, toTime, nil
}

func setSSEHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()
}
