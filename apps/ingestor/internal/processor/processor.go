package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"miniot/apps/ingestor/internal/storage"
	"miniot/apps/ingestor/internal/validator"
)

const (
	ProcessTimeout = 5 * time.Second
)

type TelemetryProcessor struct {
	storage   *storage.Storage
	validator *validator.TelemetryValidator
}

// NewTelemetryProcessor creates an instance of TelemetryProcessor with precompiled schema validation.
func NewTelemetryProcessor(s *storage.Storage) (*TelemetryProcessor, error) {
	v, err := validator.NewTelemetryValidator()
	if err != nil {
		return nil, fmt.Errorf("failed initializing validator: %w", err)
	}

	return &TelemetryProcessor{
		storage:   s,
		validator: v,
	}, nil
}

// ProcessMessage validates incoming MQTT telemetry and persists to TimescaleDB.
func (p *TelemetryProcessor) ProcessMessage(topic string, payload []byte) error {
	receivedAt := time.Now().UTC()

	validated, err := p.validator.ValidateAndParse(topic, payload, receivedAt)
	if err != nil {
		log.Printf("[Processor WARN] Rejected malformed message on %s: %v", topic, err)
		return fmt.Errorf("telemetry validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ProcessTimeout)
	defer cancel()

	if err := p.storage.SaveTelemetry(ctx, validated); err != nil {
		log.Printf("[Processor ERROR] Storage persistence failed for %s: %v", validated.DeviceID, err)
		return fmt.Errorf("storage save failed: %w", err)
	}

	return nil
}

// Validator returns the underlying telemetry validator.
func (p *TelemetryProcessor) Validator() *validator.TelemetryValidator {
	return p.validator
}

// Storage returns the underlying storage handle.
func (p *TelemetryProcessor) Storage() *storage.Storage {
	return p.storage
}
