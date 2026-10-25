package models

import "time"

type CustomWidget struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	WidgetType     string                 `json:"widget_type"`
	TemplateHTML   string                 `json:"template_html"`
	StyleCSS       string                 `json:"style_css"`
	ControllerJS   string                 `json:"controller_js"`
	SettingsSchema map[string]interface{} `json:"settings_schema"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type CreateCustomWidgetRequest struct {
	Name           string                 `json:"name" binding:"required,min=1,max=255"`
	Description    string                 `json:"description"`
	WidgetType     string                 `json:"widget_type"`
	TemplateHTML   string                 `json:"template_html"`
	StyleCSS       string                 `json:"style_css"`
	ControllerJS   string                 `json:"controller_js"`
	SettingsSchema map[string]interface{} `json:"settings_schema"`
}

type UpdateCustomWidgetRequest struct {
	Name           *string                `json:"name"`
	Description    *string                `json:"description"`
	WidgetType     *string                `json:"widget_type"`
	TemplateHTML   *string                `json:"template_html"`
	StyleCSS       *string                `json:"style_css"`
	ControllerJS   *string                `json:"controller_js"`
	SettingsSchema map[string]interface{} `json:"settings_schema"`
}
