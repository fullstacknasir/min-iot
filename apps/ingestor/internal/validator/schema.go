package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	SchemaResourceURL           = "https://miniot.io/schemas/telemetry.json"
	TopicTelemetryPrefix        = "devices/"
	TopicTelemetrySuffix        = "/telemetry"
	TopicSegmentCount           = 3
	DeviceIDSegmentIndex        = 1
	UnixMillisThreshold float64 = 1e11
	RFC3339LayoutNoSeconds      = "2006-01-02T15:04Z07:00"
	StandardDateTimeLayout      = "2006-01-02 15:04:05"

	TelemetryJSONSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "DeviceTelemetryPayload",
  "type": "object",
  "required": ["timestamp"],
  "properties": {
    "device_id": {
      "type": "string",
      "minLength": 1
    },
    "timestamp": {
      "description": "Device timestamp in ISO 8601 string or numeric Unix epoch",
      "oneOf": [
        {"type": "string", "minLength": 1},
        {"type": "number", "minimum": 0}
      ]
    },
    "metrics": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "number"
      }
    }
  }
}`
)

var (
	ErrMalformedJSON          = errors.New("malformed JSON message")
	ErrInvalidTopicFormat     = errors.New("invalid MQTT telemetry topic format")
	ErrDeviceIDMismatch       = errors.New("device_id in payload does not match topic device_id")
	ErrInvalidDeviceTimestamp = errors.New("invalid or unparseable device timestamp")
	ErrMissingMetrics         = errors.New("telemetry must include at least one valid numeric metric")
	ErrInvalidMetricValue     = errors.New("metric value must be a valid finite number")
)

type ValidatedTelemetry struct {
	DeviceID   string
	DeviceTime time.Time
	ReceivedAt time.Time
	Metrics    map[string]float64
}

type TelemetryValidator struct {
	compiledSchema *jsonschema.Schema
}

// NewTelemetryValidator initializes and pre-compiles the JSON Schema validator.
func NewTelemetryValidator() (*TelemetryValidator, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(SchemaResourceURL, strings.NewReader(TelemetryJSONSchema)); err != nil {
		return nil, fmt.Errorf("failed adding schema resource: %w", err)
	}

	schema, err := compiler.Compile(SchemaResourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed compiling JSON schema: %w", err)
	}

	return &TelemetryValidator{compiledSchema: schema}, nil
}

// ValidateAndParse validates the incoming MQTT topic and JSON payload strictly.
func (v *TelemetryValidator) ValidateAndParse(topic string, payload []byte, receivedAt time.Time) (*ValidatedTelemetry, error) {
	topicDeviceID, err := ExtractDeviceIDFromTopic(topic)
	if err != nil {
		return nil, err
	}

	rawMap, err := parseJSONMap(payload)
	if err != nil {
		return nil, err
	}

	if err := v.validateSchema(rawMap); err != nil {
		return nil, err
	}

	if err := verifyDeviceIDMatch(topicDeviceID, rawMap); err != nil {
		return nil, err
	}

	deviceTime, err := parseDeviceTimestamp(rawMap["timestamp"])
	if err != nil {
		return nil, err
	}

	metrics, err := extractMetrics(rawMap)
	if err != nil {
		return nil, err
	}

	return &ValidatedTelemetry{
		DeviceID:   topicDeviceID,
		DeviceTime: deviceTime,
		ReceivedAt: receivedAt.UTC(),
		Metrics:    metrics,
	}, nil
}

// ExtractDeviceIDFromTopic extracts the device ID from topic pattern devices/{deviceId}/telemetry.
func ExtractDeviceIDFromTopic(topic string) (string, error) {
	trimmed := strings.TrimSpace(topic)
	if !strings.HasPrefix(trimmed, TopicTelemetryPrefix) || !strings.HasSuffix(trimmed, TopicTelemetrySuffix) {
		return "", fmt.Errorf("%w: expected devices/{deviceId}/telemetry, got %s", ErrInvalidTopicFormat, topic)
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) != TopicSegmentCount || parts[DeviceIDSegmentIndex] == "" {
		return "", fmt.Errorf("%w: missing device ID segment in %s", ErrInvalidTopicFormat, topic)
	}

	return parts[DeviceIDSegmentIndex], nil
}

func parseJSONMap(payload []byte) (map[string]interface{}, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("%w: payload is empty", ErrMalformedJSON)
	}

	var data interface{}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: root value must be an object", ErrMalformedJSON)
	}

	return dataMap, nil
}

func (v *TelemetryValidator) validateSchema(rawMap map[string]interface{}) error {
	normalizedMap := normalizeMapForValidation(rawMap)
	if err := v.compiledSchema.Validate(normalizedMap); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	return nil
}

func normalizeMapForValidation(rawMap map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(rawMap))
	for key, val := range rawMap {
		normalized[key] = normalizeValue(val)
	}
	return normalized
}

func normalizeValue(val interface{}) interface{} {
	switch v := val.(type) {
	case json.Number:
		if numFloat, err := v.Float64(); err == nil {
			return numFloat
		}
		return v.String()
	case map[string]interface{}:
		return normalizeMapForValidation(v)
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, item := range v {
			res[i] = normalizeValue(item)
		}
		return res
	default:
		return v
	}
}

func verifyDeviceIDMatch(topicDeviceID string, rawMap map[string]interface{}) error {
	payloadDeviceID, exists := rawMap["device_id"]
	if !exists || payloadDeviceID == nil {
		return nil
	}

	payloadDeviceIDStr, ok := payloadDeviceID.(string)
	if !ok || strings.TrimSpace(payloadDeviceIDStr) == "" {
		return fmt.Errorf("%w: payload device_id must be a non-empty string", ErrMalformedJSON)
	}

	if payloadDeviceIDStr != topicDeviceID {
		return fmt.Errorf("%w: topic has '%s' but payload has '%s'", ErrDeviceIDMismatch, topicDeviceID, payloadDeviceIDStr)
	}

	return nil
}

func parseDeviceTimestamp(raw interface{}) (time.Time, error) {
	if raw == nil {
		return time.Time{}, ErrInvalidDeviceTimestamp
	}

	switch v := raw.(type) {
	case string:
		return parseTimestampString(v)
	case json.Number:
		num, err := v.Float64()
		if err != nil {
			return time.Time{}, ErrInvalidDeviceTimestamp
		}
		return parseTimestampNumber(num)
	case float64:
		return parseTimestampNumber(v)
	default:
		return time.Time{}, ErrInvalidDeviceTimestamp
	}
}

func parseTimestampString(str string) (time.Time, error) {
	cleaned := strings.TrimSpace(str)
	if cleaned == "" {
		return time.Time{}, ErrInvalidDeviceTimestamp
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		RFC3339LayoutNoSeconds,
		StandardDateTimeLayout,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, cleaned); err == nil {
			return parsed.UTC(), nil
		}
	}

	if epochNum, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return parseTimestampNumber(epochNum)
	}

	return time.Time{}, fmt.Errorf("%w: unrecognized date format '%s'", ErrInvalidDeviceTimestamp, cleaned)
}

func parseTimestampNumber(num float64) (time.Time, error) {
	if num <= 0 || math.IsNaN(num) || math.IsInf(num, 0) {
		return time.Time{}, fmt.Errorf("%w: timestamp number must be positive and finite", ErrInvalidDeviceTimestamp)
	}

	if num >= UnixMillisThreshold {
		return time.UnixMilli(int64(num)).UTC(), nil
	}

	sec := int64(num)
	nsec := int64((num - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC(), nil
}

func extractMetrics(rawMap map[string]interface{}) (map[string]float64, error) {
	if rawMetrics, hasMetrics := rawMap["metrics"]; hasMetrics {
		return parseMetricsObject(rawMetrics)
	}

	return extractFlatMetrics(rawMap)
}

func parseMetricsObject(raw interface{}) (map[string]float64, error) {
	metricsMap, ok := raw.(map[string]interface{})
	if !ok || len(metricsMap) == 0 {
		return nil, ErrMissingMetrics
	}

	result := make(map[string]float64, len(metricsMap))
	for name, val := range metricsMap {
		numVal, err := toFiniteFloat64(val)
		if err != nil {
			return nil, fmt.Errorf("%w: metric '%s'", err, name)
		}
		result[name] = numVal
	}

	return result, nil
}

func extractFlatMetrics(rawMap map[string]interface{}) (map[string]float64, error) {
	result := make(map[string]float64)
	for key, val := range rawMap {
		if key == "device_id" || key == "timestamp" || key == "ts" {
			continue
		}
		numVal, err := toFiniteFloat64(val)
		if err != nil {
			return nil, fmt.Errorf("%w: property '%s' is not a valid number", ErrInvalidMetricValue, key)
		}
		result[key] = numVal
	}

	if len(result) == 0 {
		return nil, ErrMissingMetrics
	}

	return result, nil
}

func toFiniteFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case json.Number:
		num, err := v.Float64()
		if err != nil {
			return 0, ErrInvalidMetricValue
		}
		return validateFinite(num)
	case float64:
		return validateFinite(v)
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, ErrInvalidMetricValue
	}
}

func validateFinite(num float64) (float64, error) {
	if math.IsNaN(num) || math.IsInf(num, 0) {
		return 0, ErrInvalidMetricValue
	}
	return num, nil
}
