package models

import (
	"time"
)

type TelemetryDataPoint struct {
	Time        time.Time  `json:"time"`
	ReceivedAt  *time.Time `json:"received_at,omitempty"`
	DeviceID    string     `json:"device_id"`
	MetricName  string     `json:"metric_name"`
	MetricValue float64    `json:"metric_value"`
	StringValue *string    `json:"string_value,omitempty"`
}

type TelemetryAggregatePoint struct {
	Bucket     time.Time `json:"bucket"`
	DeviceID   string    `json:"device_id"`
	MetricName string    `json:"metric_name"`
	AvgValue   float64   `json:"avg_value"`
	MinValue   float64   `json:"min_value"`
	MaxValue   float64   `json:"max_value"`
	CountValue int64     `json:"count_value"`
}

type LatestTelemetryData struct {
	DeviceID   string                 `json:"device_id"`
	Timestamp  *time.Time             `json:"timestamp,omitempty"`
	ReceivedAt *time.Time             `json:"received_at,omitempty"`
	Metrics    map[string]interface{} `json:"metrics"`
	Status     string                 `json:"status,omitempty"`
	UpdatedAt  *time.Time             `json:"updated_at,omitempty"`
}

type TelemetryHistoryResponse struct {
	DeviceID string               `json:"device_id"`
	Count    int                  `json:"count"`
	Data     []TelemetryDataPoint `json:"data"`
}

type TelemetryAggregatesResponse struct {
	DeviceID   string                    `json:"device_id"`
	Bucket     string                    `json:"bucket"`
	Count      int                       `json:"count"`
	Aggregates []TelemetryAggregatePoint `json:"aggregates"`
}

type IngestTelemetryRequest struct {
	DeviceID  string             `json:"device_id" binding:"required"`
	Timestamp *time.Time         `json:"timestamp"`
	Metrics   map[string]float64 `json:"metrics" binding:"required"`
}
