package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"miniot/apps/api/internal/database"
	"miniot/apps/api/internal/models"
)

const HealthCheckTimeout = 2 * time.Second

type HealthHandler struct {
	db *database.Clients
}

func NewHealthHandler(db *database.Clients) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check verifies database and cache connectivity and reports platform health.
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), HealthCheckTimeout)
	defer cancel()

	services := make(map[string]string)
	status := "healthy"

	if h.db == nil || h.db.Pool == nil {
		services["timescaledb"] = "uninitialized"
		status = "degraded"
	} else if err := h.db.Pool.Ping(ctx); err != nil {
		services["timescaledb"] = "unreachable"
		status = "degraded"
	} else {
		services["timescaledb"] = "connected"
	}

	if h.db == nil || h.db.Redis == nil {
		services["redis"] = "uninitialized"
		status = "degraded"
	} else if err := h.db.Redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "unreachable"
		status = "degraded"
	} else {
		services["redis"] = "connected"
	}

	code := http.StatusOK
	if status != "healthy" {
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, models.SystemHealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Services:  services,
	})
}
