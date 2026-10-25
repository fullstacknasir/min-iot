package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
)

const (
	DefaultRuleCooldownSeconds = 60
	DefaultRuleSeverity        = models.SeverityWarning
	DefaultRuleChannel         = models.ChannelInApp
	NotificationSSEKeepalive   = 15 * time.Second
)

type RuleAlertHandler struct {
	repo  *repository.RuleAlertRepository
	redis *redis.Client
}

func NewRuleAlertHandler(repo *repository.RuleAlertRepository, rdb *redis.Client) *RuleAlertHandler {
	return &RuleAlertHandler{repo: repo, redis: rdb}
}

// ---------------------------------------------------------------------------
// Rules Endpoints
// ---------------------------------------------------------------------------

func (h *RuleAlertHandler) ListRules(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	page, limit := parsePaginationParams(c)
	deviceID := c.Query("device_id")

	var enabledPtr *bool
	if enStr := c.Query("enabled"); enStr != "" {
		isEn := enStr == "true"
		enabledPtr = &isEn
	}

	rules, total, err := h.repo.ListRules(c.Request.Context(), orgID, page, limit, deviceID, enabledPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = total
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        rules,
		"total":       total,
		"page":        page,
		"limit":       effectiveLimit,
		"total_pages": models.ComputeTotalPages(total, effectiveLimit),
	})
}

func (h *RuleAlertHandler) GetRule(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	ruleID := c.Param("id")

	rule, err := h.repo.GetRuleByID(c.Request.Context(), ruleID, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

func (h *RuleAlertHandler) CreateRule(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	var req models.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !models.IsValidCondition(req.Condition) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid condition operator; must be >, >=, <, <=, ==, !="})
		return
	}

	severity := req.Severity
	if severity == "" {
		severity = DefaultRuleSeverity
	}
	if !models.IsValidSeverity(severity) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid severity; must be critical, warning, or info"})
		return
	}

	cooldown := DefaultRuleCooldownSeconds
	if req.CooldownSeconds != nil && *req.CooldownSeconds >= 0 {
		cooldown = *req.CooldownSeconds
	}

	channel := req.NotificationChannel
	if channel == "" {
		channel = DefaultRuleChannel
	}
	if !models.IsValidNotificationChannel(channel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification channel"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &models.Rule{
		ID:                  "rule_" + uuid.New().String()[:12],
		OrganizationID:      orgID,
		Name:                req.Name,
		Description:         req.Description,
		DeviceID:            req.DeviceID,
		DeviceType:          req.DeviceType,
		MetricName:          req.MetricName,
		Condition:           req.Condition,
		Threshold:           req.Threshold,
		Severity:            severity,
		CooldownSeconds:     cooldown,
		NotificationChannel: channel,
		WebhookURL:          req.WebhookURL,
		Enabled:             enabled,
	}

	if err := h.repo.CreateRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

func (h *RuleAlertHandler) UpdateRule(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	ruleID := c.Param("id")

	rule, err := h.repo.GetRuleByID(c.Request.Context(), ruleID, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req models.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.applyRuleUpdates(rule, &req)

	if err := h.repo.UpdateRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

func (h *RuleAlertHandler) applyRuleUpdates(rule *models.Rule, req *models.UpdateRuleRequest) {
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.ClearDeviceID {
		rule.DeviceID = nil
	} else if req.DeviceID != nil {
		rule.DeviceID = req.DeviceID
	}
	if req.DeviceType != nil {
		rule.DeviceType = *req.DeviceType
	}
	if req.MetricName != nil {
		rule.MetricName = *req.MetricName
	}
	if req.Condition != nil && models.IsValidCondition(*req.Condition) {
		rule.Condition = *req.Condition
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.Severity != nil && models.IsValidSeverity(*req.Severity) {
		rule.Severity = *req.Severity
	}
	if req.CooldownSeconds != nil && *req.CooldownSeconds >= 0 {
		rule.CooldownSeconds = *req.CooldownSeconds
	}
	if req.NotificationChannel != nil && models.IsValidNotificationChannel(*req.NotificationChannel) {
		rule.NotificationChannel = *req.NotificationChannel
	}
	if req.WebhookURL != nil {
		rule.WebhookURL = *req.WebhookURL
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
}

func (h *RuleAlertHandler) DeleteRule(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	ruleID := c.Param("id")

	if err := h.repo.DeleteRule(c.Request.Context(), ruleID, orgID); err != nil {
		if errors.Is(err, repository.ErrRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted successfully"})
}

func (h *RuleAlertHandler) ToggleRule(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	ruleID := c.Param("id")

	newStatus, err := h.repo.ToggleRule(c.Request.Context(), ruleID, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enabled": newStatus})
}

// ---------------------------------------------------------------------------
// Alerts Endpoints
// ---------------------------------------------------------------------------

func (h *RuleAlertHandler) ListAlerts(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	page, limit := parsePaginationParams(c)
	status := c.Query("status")
	severity := c.Query("severity")
	deviceID := c.Query("device_id")

	alerts, total, err := h.repo.ListAlerts(c.Request.Context(), orgID, page, limit, status, severity, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = total
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        alerts,
		"total":       total,
		"page":        page,
		"limit":       effectiveLimit,
		"total_pages": models.ComputeTotalPages(total, effectiveLimit),
	})
}

func (h *RuleAlertHandler) GetAlert(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	alertID := c.Param("id")

	alert, err := h.repo.GetAlertByID(c.Request.Context(), alertID, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrAlertNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alert})
}

func (h *RuleAlertHandler) AcknowledgeAlert(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	userID := middleware.GetUserID(c)
	alertID := c.Param("id")

	if err := h.repo.AcknowledgeAlert(c.Request.Context(), alertID, orgID, userID); err != nil {
		if errors.Is(err, repository.ErrAlertNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "active alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert acknowledged", "alert_id": alertID})
}

func (h *RuleAlertHandler) ResolveAlert(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	userID := middleware.GetUserID(c)
	alertID := c.Param("id")

	if err := h.repo.ResolveAlert(c.Request.Context(), alertID, orgID, userID); err != nil {
		if errors.Is(err, repository.ErrAlertNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unresolved alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert resolved", "alert_id": alertID})
}

// ---------------------------------------------------------------------------
// Notifications Endpoints
// ---------------------------------------------------------------------------

func (h *RuleAlertHandler) ListNotifications(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	page, limit := parsePaginationParams(c)
	unreadOnly := c.Query("unread") == "true"

	notifications, total, err := h.repo.ListNotifications(c.Request.Context(), orgID, page, limit, unreadOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = total
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        notifications,
		"total":       total,
		"page":        page,
		"limit":       effectiveLimit,
		"total_pages": models.ComputeTotalPages(total, effectiveLimit),
	})
}

func (h *RuleAlertHandler) GetUnreadCount(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	count, err := h.repo.GetUnreadNotificationCount(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

func (h *RuleAlertHandler) MarkAsRead(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	notificationID := c.Param("id")

	if err := h.repo.MarkNotificationAsRead(c.Request.Context(), notificationID, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

func (h *RuleAlertHandler) MarkAllAsRead(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	if err := h.repo.MarkAllNotificationsAsRead(c.Request.Context(), orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

func (h *RuleAlertHandler) StreamLiveNotifications(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	if h.redis == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "real-time notification service unavailable"})
		return
	}

	setSSEHeaders(c)

	channelName := fmt.Sprintf("org:%s:notifications", orgID)
	pubsub := h.redis.Subscribe(c.Request.Context(), channelName)
	defer pubsub.Close()

	ch := pubsub.Channel()
	ticker := time.NewTicker(NotificationSSEKeepalive)
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
			fmt.Fprintf(w, "event: notification\ndata: %s\n\n", msg.Payload)
			w.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			w.Flush()
		}
	}
}
