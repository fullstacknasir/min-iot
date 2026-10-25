package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
)

const (
	DefaultWebhookTimeout = 5 * time.Second
	EpsilonTolerance      = 1e-9
)

type RuleEngine struct {
	repo       *repository.RuleAlertRepository
	deviceRepo *repository.DeviceRepository
	redis      *redis.Client
	httpClient *http.Client
}

type TriggeredEventPayload struct {
	Alert        *models.Alert        `json:"alert"`
	Notification *models.Notification `json:"notification"`
}

func NewRuleEngine(repo *repository.RuleAlertRepository, deviceRepo *repository.DeviceRepository, rdb *redis.Client) *RuleEngine {
	return &RuleEngine{
		repo:       repo,
		deviceRepo: deviceRepo,
		redis:      rdb,
		httpClient: &http.Client{Timeout: DefaultWebhookTimeout},
	}
}

// EvaluateMetric checks all active rules for the given device and metric value.
func (e *RuleEngine) EvaluateMetric(ctx context.Context, orgID, deviceID, deviceType, metricName string, metricValue float64, triggeredAt time.Time) ([]*models.Alert, error) {
	rules, err := e.repo.FindActiveRulesForDevice(ctx, orgID, deviceID, deviceType, metricName)
	if err != nil || len(rules) == 0 {
		return nil, err
	}

	var generatedAlerts []*models.Alert
	for _, rule := range rules {
		if !EvaluateCondition(rule.Condition, metricValue, rule.Threshold) {
			continue
		}

		if e.isCoolingDown(ctx, rule.ID, deviceID) {
			continue
		}

		alert, notification, err := e.triggerRule(ctx, &rule, orgID, deviceID, metricName, metricValue, triggeredAt)
		if err != nil {
			log.Printf("[RuleEngine ERROR] Failed triggering rule %s: %v", rule.ID, err)
			continue
		}

		generatedAlerts = append(generatedAlerts, alert)
		e.dispatchNotification(ctx, &rule, alert, notification)
	}

	return generatedAlerts, nil
}

// EvaluateBatch processes multiple metrics for a single device.
func (e *RuleEngine) EvaluateBatch(ctx context.Context, orgID, deviceID string, metrics map[string]float64, timestamp time.Time) error {
	device, err := e.deviceRepo.FindByID(ctx, deviceID, orgID)
	deviceType := ""
	if err == nil && device != nil {
		deviceType = device.DeviceType
	}

	for metricName, metricVal := range metrics {
		_, _ = e.EvaluateMetric(ctx, orgID, deviceID, deviceType, metricName, metricVal, timestamp)
	}
	return nil
}

// EvaluateCondition evaluates numeric comparison operators.
func EvaluateCondition(condition string, actualValue, threshold float64) bool {
	switch condition {
	case models.ConditionGT:
		return actualValue > threshold
	case models.ConditionGTE:
		return actualValue >= threshold
	case models.ConditionLT:
		return actualValue < threshold
	case models.ConditionLTE:
		return actualValue <= threshold
	case models.ConditionEQ:
		return math.Abs(actualValue-threshold) < EpsilonTolerance
	case models.ConditionNEQ:
		return math.Abs(actualValue-threshold) >= EpsilonTolerance
	default:
		return false
	}
}

func (e *RuleEngine) isCoolingDown(ctx context.Context, ruleID, deviceID string) bool {
	if e.redis == nil {
		return false
	}
	cooldownKey := fmt.Sprintf("cooldown:rule:%s:device:%s", ruleID, deviceID)
	exists, err := e.redis.Exists(ctx, cooldownKey).Result()
	return err == nil && exists > 0
}

func (e *RuleEngine) setCooldown(ctx context.Context, ruleID, deviceID string, seconds int) {
	if e.redis == nil || seconds <= 0 {
		return
	}
	cooldownKey := fmt.Sprintf("cooldown:rule:%s:device:%s", ruleID, deviceID)
	_ = e.redis.Set(ctx, cooldownKey, "1", time.Duration(seconds)*time.Second).Err()
}

func (e *RuleEngine) triggerRule(
	ctx context.Context,
	rule *models.Rule,
	orgID, deviceID, metricName string,
	metricValue float64,
	triggeredAt time.Time,
) (*models.Alert, *models.Notification, error) {
	alertID := "alt_" + uuid.New().String()[:12]
	notificationID := "ntf_" + uuid.New().String()[:12]

	msg := fmt.Sprintf("%s (%.2f %s %.2f) detected on device %s",
		rule.Name, metricValue, rule.Condition, rule.Threshold, deviceID)

	alert := &models.Alert{
		ID:             alertID,
		OrganizationID: orgID,
		RuleID:         &rule.ID,
		RuleName:       &rule.Name,
		DeviceID:       deviceID,
		Severity:       rule.Severity,
		Status:         models.AlertStatusActive,
		Message:        msg,
		MetricName:     metricName,
		MetricValue:    metricValue,
		ThresholdValue: rule.Threshold,
		TriggeredAt:    triggeredAt,
	}

	if err := e.repo.CreateAlert(ctx, alert); err != nil {
		return nil, nil, fmt.Errorf("failed persisting alert: %w", err)
	}

	notification := &models.Notification{
		ID:             notificationID,
		OrganizationID: orgID,
		AlertID:        &alertID,
		Title:          fmt.Sprintf("[%s] %s", strings.ToUpper(rule.Severity), rule.Name),
		Message:        msg,
		Severity:       rule.Severity,
		Channel:        rule.NotificationChannel,
		Read:           false,
		CreatedAt:      time.Now().UTC(),
	}

	if err := e.repo.CreateNotification(ctx, notification); err != nil {
		log.Printf("[RuleEngine WARN] Notification creation failed: %v", err)
	}

	e.setCooldown(ctx, rule.ID, deviceID, rule.CooldownSeconds)
	return alert, notification, nil
}

func (e *RuleEngine) dispatchNotification(
	ctx context.Context,
	rule *models.Rule,
	alert *models.Alert,
	notification *models.Notification,
) {
	if e.redis != nil {
		payload := TriggeredEventPayload{
			Alert:        alert,
			Notification: notification,
		}
		if encoded, err := json.Marshal(payload); err == nil {
			channel := fmt.Sprintf("org:%s:notifications", alert.OrganizationID)
			_ = e.redis.Publish(ctx, channel, encoded).Err()
		}
	}

	if (rule.NotificationChannel == models.ChannelWebhook || rule.NotificationChannel == models.ChannelAll) && rule.WebhookURL != "" {
		go e.sendWebhook(rule.WebhookURL, alert, notification)
	}
}

func (e *RuleEngine) sendWebhook(url string, alert *models.Alert, notification *models.Notification) {
	payload := map[string]interface{}{
		"event":        "alert_triggered",
		"alert":        alert,
		"notification": notification,
		"timestamp":    time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "minIoT-RuleEngine/1.0")

	resp, err := e.httpClient.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}
