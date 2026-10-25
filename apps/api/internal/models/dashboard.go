package models

import (
	"encoding/json"
	"time"
)

type WidgetConfig struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Title      string                 `json:"title"`
	DeviceID   string                 `json:"device_id"`
	DeviceName string                 `json:"device_name,omitempty"`
	MetricName string                 `json:"metric_name,omitempty"`
	ColSpan    int                    `json:"col_span"`
	RowSpan    int                    `json:"row_span"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
}

type DashboardConfiguration struct {
	Widgets []WidgetConfig `json:"widgets"`
}

type Dashboard struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	IsDefault      bool                   `json:"is_default"`
	Configuration  DashboardConfiguration `json:"configuration"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type CreateDashboardRequest struct {
	Title         string                  `json:"title" binding:"required,min=1,max=255"`
	Description   string                  `json:"description"`
	IsDefault     bool                    `json:"is_default"`
	Configuration *DashboardConfiguration `json:"configuration"`
}

type UpdateDashboardRequest struct {
	Title         *string                 `json:"title"`
	Description   *string                 `json:"description"`
	IsDefault     *bool                   `json:"is_default"`
	Configuration *DashboardConfiguration `json:"configuration"`
}

// EnsureValidConfiguration guarantees non-nil widget slice
func (c *DashboardConfiguration) EnsureValidConfiguration() {
	if c.Widgets == nil {
		c.Widgets = make([]WidgetConfig, 0)
	}
}

// ToJSON converts configuration to JSON bytes
func (c *DashboardConfiguration) ToJSON() ([]byte, error) {
	c.EnsureValidConfiguration()
	return json.Marshal(c)
}
