package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
)

type CustomWidgetHandler struct {
	repo *repository.CustomWidgetRepository
}

func NewCustomWidgetHandler(repo *repository.CustomWidgetRepository) *CustomWidgetHandler {
	return &CustomWidgetHandler{repo: repo}
}

// ListCustomWidgets returns all custom widgets in the user's organization.
func (h *CustomWidgetHandler) ListCustomWidgets(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	widgets, err := h.repo.ListCustomWidgets(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed retrieving custom widgets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  widgets,
		"count": len(widgets),
	})
}

// GetCustomWidget returns a custom widget by its ID.
func (h *CustomWidgetHandler) GetCustomWidget(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	id := c.Param("id")

	if orgID == "" || id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or widget identifier"})
		return
	}

	widget, err := h.repo.GetCustomWidgetByID(c.Request.Context(), id, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrCustomWidgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "custom widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed retrieving custom widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": widget})
}

// CreateCustomWidget saves a new custom widget definition.
func (h *CustomWidgetHandler) CreateCustomWidget(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	var req models.CreateCustomWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	widgetType := req.WidgetType
	if widgetType == "" {
		widgetType = "telemetry"
	}

	schema := req.SettingsSchema
	if schema == nil {
		schema = make(map[string]interface{})
	}

	widget := &models.CustomWidget{
		ID:             "cw-" + uuid.New().String()[:8],
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		WidgetType:     widgetType,
		TemplateHTML:   req.TemplateHTML,
		StyleCSS:       req.StyleCSS,
		ControllerJS:   req.ControllerJS,
		SettingsSchema: schema,
	}

	if err := h.repo.CreateCustomWidget(c.Request.Context(), widget); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating custom widget"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "custom widget created successfully",
		"data":    widget,
	})
}

// UpdateCustomWidget updates an existing custom widget definition.
func (h *CustomWidgetHandler) UpdateCustomWidget(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	id := c.Param("id")

	if orgID == "" || id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or widget identifier"})
		return
	}

	existing, err := h.repo.GetCustomWidgetByID(c.Request.Context(), id, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrCustomWidgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "custom widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed retrieving custom widget"})
		return
	}

	var req models.UpdateCustomWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.WidgetType != nil {
		existing.WidgetType = *req.WidgetType
	}
	if req.TemplateHTML != nil {
		existing.TemplateHTML = *req.TemplateHTML
	}
	if req.StyleCSS != nil {
		existing.StyleCSS = *req.StyleCSS
	}
	if req.ControllerJS != nil {
		existing.ControllerJS = *req.ControllerJS
	}
	if req.SettingsSchema != nil {
		existing.SettingsSchema = req.SettingsSchema
	}

	if err := h.repo.UpdateCustomWidget(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed updating custom widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "custom widget updated successfully",
		"data":    existing,
	})
}

// DeleteCustomWidget deletes a custom widget definition.
func (h *CustomWidgetHandler) DeleteCustomWidget(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	id := c.Param("id")

	if orgID == "" || id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or widget identifier"})
		return
	}

	if err := h.repo.DeleteCustomWidget(c.Request.Context(), id, orgID); err != nil {
		if errors.Is(err, repository.ErrCustomWidgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "custom widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed deleting custom widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "custom widget deleted successfully"})
}
