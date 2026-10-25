package models

import (
	"time"
)

const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"

	AlertStatusActive       = "active"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusResolved     = "resolved"

	ChannelInApp   = "in_app"
	ChannelEmail   = "email"
	ChannelWebhook = "webhook"
	ChannelAll     = "all"

	ConditionGT  = ">"
	ConditionGTE = ">="
	ConditionLT  = "<"
	ConditionLTE = "<="
	ConditionEQ  = "=="
	ConditionNEQ = "!="
)

type Rule struct {
	ID                  string    `json:"id"`
	OrganizationID      string    `json:"organization_id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	DeviceID            *string   `json:"device_id,omitempty"`
	DeviceName          *string   `json:"device_name,omitempty"`
	DeviceType          string    `json:"device_type"`
	MetricName          string    `json:"metric_name"`
	Condition           string    `json:"condition"`
	Threshold           float64   `json:"threshold"`
	Severity            string    `json:"severity"`
	CooldownSeconds     int       `json:"cooldown_seconds"`
	NotificationChannel string    `json:"notification_channel"`
	WebhookURL          string    `json:"webhook_url"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateRuleRequest struct {
	Name                string   `json:"name" binding:"required,min=2,max=255"`
	Description         string   `json:"description"`
	DeviceID            *string  `json:"device_id"`
	DeviceType          string   `json:"device_type"`
	MetricName          string   `json:"metric_name" binding:"required"`
	Condition           string   `json:"condition" binding:"required"`
	Threshold           float64  `json:"threshold" binding:"required"`
	Severity            string   `json:"severity"`
	CooldownSeconds     *int     `json:"cooldown_seconds"`
	NotificationChannel string   `json:"notification_channel"`
	WebhookURL          string   `json:"webhook_url"`
	Enabled             *bool    `json:"enabled"`
}

type UpdateRuleRequest struct {
	Name                *string  `json:"name"`
	Description         *string  `json:"description"`
	DeviceID            *string  `json:"device_id"`
	ClearDeviceID       bool     `json:"clear_device_id"`
	DeviceType          *string  `json:"device_type"`
	MetricName          *string  `json:"metric_name"`
	Condition           *string  `json:"condition"`
	Threshold           *float64 `json:"threshold"`
	Severity            *string  `json:"severity"`
	CooldownSeconds     *int     `json:"cooldown_seconds"`
	NotificationChannel *string  `json:"notification_channel"`
	WebhookURL          *string  `json:"webhook_url"`
	Enabled             *bool    `json:"enabled"`
}

type Alert struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	RuleID         *string    `json:"rule_id,omitempty"`
	RuleName       *string    `json:"rule_name,omitempty"`
	DeviceID       string     `json:"device_id"`
	DeviceName     *string    `json:"device_name,omitempty"`
	Severity       string     `json:"severity"`
	Status         string     `json:"status"`
	Message        string     `json:"message"`
	MetricName     string     `json:"metric_name"`
	MetricValue    float64    `json:"metric_value"`
	ThresholdValue float64    `json:"threshold_value"`
	TriggeredAt    time.Time  `json:"triggered_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *string    `json:"acknowledged_by,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     *string    `json:"resolved_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Notification struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	AlertID        *string   `json:"alert_id,omitempty"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	Severity       string    `json:"severity"`
	Channel        string    `json:"channel"`
	Read           bool      `json:"read"`
	CreatedAt      time.Time `json:"created_at"`
}

func IsValidCondition(cond string) bool {
	switch cond {
	case ConditionGT, ConditionGTE, ConditionLT, ConditionLTE, ConditionEQ, ConditionNEQ:
		return true
	default:
		return false
	}
}

func IsValidSeverity(sev string) bool {
	switch sev {
	case SeverityCritical, SeverityWarning, SeverityInfo:
		return true
	default:
		return false
	}
}

func IsValidNotificationChannel(ch string) bool {
	switch ch {
	case ChannelInApp, ChannelEmail, ChannelWebhook, ChannelAll:
		return true
	default:
		return false
	}
}
