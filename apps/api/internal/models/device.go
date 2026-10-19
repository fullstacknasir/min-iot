package models

import "time"

const (
	DeviceStatusOnline   = "online"
	DeviceStatusOffline  = "offline"
	DeviceStatusDisabled = "disabled"
	DeviceStatusRevoked  = "revoked"

	CredentialStatusActive   = "active"
	CredentialStatusDisabled = "disabled"
	CredentialStatusRevoked  = "revoked"

	DefaultCredentialType = "token"
)

type Device struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	Name           string                 `json:"name" binding:"required"`
	DeviceType     string                 `json:"device_type" binding:"required"`
	Status         string                 `json:"status"`
	Metadata       map[string]interface{} `json:"metadata"`
	LastSeen       *time.Time             `json:"last_seen"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type CreateDeviceRequest struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name" binding:"required"`
	DeviceType string                 `json:"device_type" binding:"required"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type UpdateDeviceRequest struct {
	Name       *string                `json:"name"`
	DeviceType *string                `json:"device_type"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type DeviceCredential struct {
	ID             string     `json:"id"`
	DeviceID       string     `json:"device_id"`
	OrganizationID string     `json:"organization_id"`
	CredentialType string     `json:"credential_type"`
	SecretHash     string     `json:"-"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type ProvisionDeviceRequest struct {
	CredentialType string `json:"credential_type"`
}

type ProvisionDeviceResponse struct {
	DeviceID       string    `json:"device_id"`
	OrganizationID string    `json:"organization_id"`
	CredentialID   string    `json:"credential_id"`
	CredentialType string    `json:"credential_type"`
	Token          string    `json:"token"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	Notice         string    `json:"notice"`
}

type DeviceStatusActionResponse struct {
	DeviceID string    `json:"device_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SystemHealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}
