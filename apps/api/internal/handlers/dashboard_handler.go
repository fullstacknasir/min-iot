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

type DashboardHandler struct {
	repo *repository.DashboardRepository
}

func NewDashboardHandler(repo *repository.DashboardRepository) *DashboardHandler {
	return &DashboardHandler{repo: repo}
}

// ListDashboards returns all dashboards belonging to the user's organization.
func (h *DashboardHandler) ListDashboards(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	dashboards, err := h.repo.ListDashboards(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed retrieving dashboards"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  dashboards,
		"count": len(dashboards),
	})
}

// GetDashboard returns a single dashboard by ID.
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	dashboardID := c.Param("id")

	if orgID == "" || dashboardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or dashboard identifier"})
		return
	}

	dashboard, err := h.repo.GetDashboardByID(c.Request.Context(), dashboardID, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrDashboardNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed retrieving dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dashboard})
}

// CreateDashboard handles creating a new customizable dashboard.
func (h *DashboardHandler) CreateDashboard(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	var req models.CreateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := models.DashboardConfiguration{}
	if req.Configuration != nil {
		cfg = *req.Configuration
	}
	cfg.EnsureValidConfiguration()

	dashboard := &models.Dashboard{
		ID:             "dash-" + uuid.New().String()[:8],
		OrganizationID: orgID,
		Title:          req.Title,
		Description:    req.Description,
		IsDefault:      req.IsDefault,
		Configuration:  cfg,
	}

	if err := h.repo.CreateDashboard(c.Request.Context(), dashboard); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating dashboard"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "dashboard created successfully",
		"data":    dashboard,
	})
}

// UpdateDashboard modifies dashboard properties and widget configuration.
func (h *DashboardHandler) UpdateDashboard(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	dashboardID := c.Param("id")

	if orgID == "" || dashboardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or dashboard identifier"})
		return
	}

	existing, err := h.repo.GetDashboardByID(c.Request.Context(), dashboardID, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrDashboardNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed locating dashboard"})
		return
	}

	var req models.UpdateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != nil && *req.Title != "" {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.IsDefault != nil {
		existing.IsDefault = *req.IsDefault
	}
	if req.Configuration != nil {
		req.Configuration.EnsureValidConfiguration()
		existing.Configuration = *req.Configuration
	}

	if err := h.repo.UpdateDashboard(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed updating dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "dashboard updated successfully",
		"data":    existing,
	})
}

// DeleteDashboard removes a dashboard.
func (h *DashboardHandler) DeleteDashboard(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	dashboardID := c.Param("id")

	if orgID == "" || dashboardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or dashboard identifier"})
		return
	}

	if err := h.repo.DeleteDashboard(c.Request.Context(), dashboardID, orgID); err != nil {
		if errors.Is(err, repository.ErrDashboardNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed deleting dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dashboard deleted successfully"})
}

// SetDefault designates the target dashboard as the organization default.
func (h *DashboardHandler) SetDefault(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	dashboardID := c.Param("id")

	if orgID == "" || dashboardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization or dashboard identifier"})
		return
	}

	if err := h.repo.SetDefaultDashboard(c.Request.Context(), dashboardID, orgID); err != nil {
		if errors.Is(err, repository.ErrDashboardNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed setting default dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default dashboard set successfully"})
}
