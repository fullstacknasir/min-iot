package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
)

type EMQXHandler struct {
	repo *repository.DeviceRepository
}

func NewEMQXHandler(repo *repository.DeviceRepository) *EMQXHandler {
	return &EMQXHandler{repo: repo}
}

const (
	IngestorClientIDPrefix = "miniot-ingestor"
	SystemClientIDPrefix   = "miniot-system"
)

// HandleAuth authenticates MQTT device credentials against provisioned hashes.
func (h *EMQXHandler) HandleAuth(c *gin.Context) {
	var req models.EMQXAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, models.EMQXAuthResponse{Result: models.EMQXResultDeny})
		return
	}

	if isSystemService(req.ClientID, req.Username) {
		c.JSON(http.StatusOK, models.EMQXAuthResponse{
			Result:      models.EMQXResultAllow,
			IsSuperuser: true,
		})
		return
	}

	_, allowed, err := h.repo.AuthenticateDevice(c.Request.Context(), req.ClientID, req.Username, req.Password)
	if err != nil || !allowed {
		c.JSON(http.StatusOK, models.EMQXAuthResponse{Result: models.EMQXResultDeny})
		return
	}

	c.JSON(http.StatusOK, models.EMQXAuthResponse{
		Result:      models.EMQXResultAllow,
		IsSuperuser: false,
	})
}

// HandleACL verifies that device only publishes to its telemetry topic and subscribes to its command topic.
func (h *EMQXHandler) HandleACL(c *gin.Context) {
	var req models.EMQXACLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, models.EMQXACLResponse{Result: models.EMQXResultDeny})
		return
	}

	if isSystemService(req.ClientID, req.Username) {
		c.JSON(http.StatusOK, models.EMQXACLResponse{Result: models.EMQXResultAllow})
		return
	}

	allowed, err := h.repo.AuthorizeDeviceAction(c.Request.Context(), req.ClientID, req.Username, req.Action, req.Topic)
	if err != nil || !allowed {
		c.JSON(http.StatusOK, models.EMQXACLResponse{Result: models.EMQXResultDeny})
		return
	}

	c.JSON(http.StatusOK, models.EMQXACLResponse{Result: models.EMQXResultAllow})
}

func isSystemService(clientID, username string) bool {
	return strings.HasPrefix(clientID, IngestorClientIDPrefix) ||
		strings.HasPrefix(clientID, SystemClientIDPrefix) ||
		strings.HasPrefix(username, IngestorClientIDPrefix) ||
		strings.HasPrefix(username, SystemClientIDPrefix)
}

// HandleEvents processes EMQX connect and disconnect webhook events, updating last_seen and online status.
func (h *EMQXHandler) HandleEvents(c *gin.Context) {
	var payload models.EMQXEventPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
		return
	}

	deviceID := payload.ClientID
	if deviceID == "" {
		deviceID = payload.Username
	}

	event := strings.ToLower(payload.Event)
	if strings.Contains(event, "connected") && !strings.Contains(event, "disconnected") {
		_ = h.repo.RecordDeviceConnect(c.Request.Context(), deviceID)
	} else if strings.Contains(event, "disconnected") {
		_ = h.repo.RecordDeviceDisconnect(c.Request.Context(), deviceID)
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed", "device_id": deviceID, "event": payload.Event})
}
