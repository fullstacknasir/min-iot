package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
	"miniot/apps/api/internal/security"
)

type DeviceHandler struct {
	repo *repository.DeviceRepository
}

func NewDeviceHandler(repo *repository.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

// ListDevices returns all devices strictly scoped to caller's organization.
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	devices, err := h.repo.FindAll(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if devices == nil {
		devices = []models.Device{}
	}
	c.JSON(http.StatusOK, gin.H{"data": devices, "total": len(devices)})
}

// GetDevice returns details for a device within caller's organization.
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	device, err := h.repo.FindByID(c.Request.Context(), deviceID, orgID)
	if errors.Is(err, repository.ErrDeviceNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": device})
}

// CreateDevice registers a new device bound to caller's organization.
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var req models.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	if req.ID == "" {
		req.ID = "dev-" + uuid.New().String()[:8]
	}

	device := models.Device{
		ID:             req.ID,
		OrganizationID: orgID,
		Name:           req.Name,
		DeviceType:     req.DeviceType,
		Metadata:       req.Metadata,
	}

	if err := h.repo.Create(c.Request.Context(), &device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": device})
}

// UpdateDevice updates metadata and configuration for a device.
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	var req models.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	updated, err := h.repo.Update(c.Request.Context(), deviceID, orgID, &req)
	if errors.Is(err, repository.ErrDeviceNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DeleteDevice deletes a device scoped to caller's organization.
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID is required"})
		return
	}

	orgID := middleware.GetOrganizationID(c)
	err := h.repo.Delete(c.Request.Context(), deviceID, orgID)
	if errors.Is(err, repository.ErrDeviceNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "device successfully deleted", "id": deviceID})
}

// ProvisionDevice generates per-device credentials, shown once and stored hashed.
func (h *DeviceHandler) ProvisionDevice(c *gin.Context) {
	deviceID := c.Param("id")
	orgID := middleware.GetOrganizationID(c)

	device, err := h.repo.FindByID(c.Request.Context(), deviceID, orgID)
	if errors.Is(err, repository.ErrDeviceNotFound) || device == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
		return
	}

	rawToken, tokenHash, err := security.GenerateDeviceSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed generating device credentials"})
		return
	}

	cred := &models.DeviceCredential{
		DeviceID:       deviceID,
		OrganizationID: orgID,
		CredentialType: models.DefaultCredentialType,
		SecretHash:     tokenHash,
		Status:         models.CredentialStatusActive,
	}

	if err := h.repo.StoreDeviceCredential(c.Request.Context(), cred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed storing credential record"})
		return
	}

	c.JSON(http.StatusCreated, models.ProvisionDeviceResponse{
		DeviceID:       deviceID,
		OrganizationID: orgID,
		CredentialID:   cred.ID,
		CredentialType: cred.CredentialType,
		Token:          rawToken,
		Status:         cred.Status,
		CreatedAt:      cred.CreatedAt,
		Notice:         "Store this credential securely. It will not be displayed again.",
	})
}

// GetDeviceCredentials returns credential metadata without exposing raw secret or hash.
func (h *DeviceHandler) GetDeviceCredentials(c *gin.Context) {
	deviceID := c.Param("id")
	orgID := middleware.GetOrganizationID(c)

	cred, err := h.repo.GetDeviceCredential(c.Request.Context(), deviceID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cred == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no credentials found for this device"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cred})
}

// EnableDevice sets device status to offline/enabled and activates credentials.
func (h *DeviceHandler) EnableDevice(c *gin.Context) {
	h.transitionDeviceState(c, models.DeviceStatusOffline, models.CredentialStatusActive, "device enabled successfully")
}

// DisableDevice sets device status to disabled and deactivates credentials.
func (h *DeviceHandler) DisableDevice(c *gin.Context) {
	h.transitionDeviceState(c, models.DeviceStatusDisabled, models.CredentialStatusDisabled, "device disabled successfully")
}

// RevokeDevice permanently sets device status to revoked and invalidates credentials.
func (h *DeviceHandler) RevokeDevice(c *gin.Context) {
	h.transitionDeviceState(c, models.DeviceStatusRevoked, models.CredentialStatusRevoked, "device credentials and access revoked")
}

func (h *DeviceHandler) transitionDeviceState(c *gin.Context, newStatus, credStatus, msg string) {
	deviceID := c.Param("id")
	orgID := middleware.GetOrganizationID(c)

	updated, err := h.repo.UpdateStatus(c.Request.Context(), deviceID, orgID, newStatus)
	if errors.Is(err, repository.ErrDeviceNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = h.repo.SetCredentialsStatus(c.Request.Context(), deviceID, orgID, credStatus)

	c.JSON(http.StatusOK, models.DeviceStatusActionResponse{
		DeviceID:  deviceID,
		Status:    updated.Status,
		Message:   msg,
		UpdatedAt: time.Now().UTC(),
	})
}
