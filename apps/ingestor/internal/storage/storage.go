package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"miniot/apps/ingestor/internal/config"
	"miniot/apps/ingestor/internal/validator"
)

const (
	DefaultConnectionTimeout   = 5 * time.Second
	RedisLatestTTL             = 24 * time.Hour
	RedisDeviceLatestKeyFormat = "device:%s:latest"
	DefaultQueryLimit          = 100
	ContinuousAggViewName      = "telemetry_hourly"
	RawRetentionDuration       = "30 days"
)

type Storage struct {
	Pool  *pgxpool.Pool
	Redis *redis.Client
}

type TelemetryPoint struct {
	Time        time.Time `json:"time"`
	ReceivedAt  time.Time `json:"received_at"`
	DeviceID    string    `json:"device_id"`
	MetricName  string    `json:"metric_name"`
	MetricValue float64   `json:"metric_value"`
}

type HourlyAggregate struct {
	Bucket      time.Time `json:"bucket"`
	DeviceID    string    `json:"device_id"`
	MetricName  string    `json:"metric_name"`
	AvgValue    float64   `json:"avg_value"`
	MinValue    float64   `json:"min_value"`
	MaxValue    float64   `json:"max_value"`
	CountValue  int64     `json:"count_value"`
}

type CachedTelemetry struct {
	DeviceID   string             `json:"device_id"`
	Timestamp  time.Time          `json:"timestamp"`
	ReceivedAt time.Time          `json:"received_at"`
	Metrics    map[string]float64 `json:"metrics"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// NewStorage initializes database connection pool, Redis client, and ensures hypertable schema.
func NewStorage(cfg *config.Config) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
	defer cancel()

	pool, err := initDatabasePool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	s := &Storage{Pool: pool, Redis: rdb}
	if err := s.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed ensuring TimescaleDB schema: %w", err)
	}

	return s, nil
}

// SaveTelemetry inserts time-series metrics into TimescaleDB and manages cache with idempotency.
func (s *Storage) SaveTelemetry(ctx context.Context, data *validator.ValidatedTelemetry) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.executeTelemetryInserts(ctx, tx, data); err != nil {
		return err
	}

	if err := s.updateDeviceStatus(ctx, tx, data); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	_ = s.updateRedisCacheIfNewest(ctx, data)
	return s.evaluateRules(ctx, data)
}

func (s *Storage) executeTelemetryInserts(ctx context.Context, tx pgx.Tx, data *validator.ValidatedTelemetry) error {
	query := `INSERT INTO telemetry (time, received_at, device_id, metric_name, metric_value)
	          VALUES ($1, $2, $3, $4, $5)
	          ON CONFLICT (time, device_id, metric_name) DO UPDATE
	          SET metric_value = EXCLUDED.metric_value,
	              received_at = LEAST(telemetry.received_at, EXCLUDED.received_at)`

	for name, val := range data.Metrics {
		if _, err := tx.Exec(ctx, query, data.DeviceTime, data.ReceivedAt, data.DeviceID, name, val); err != nil {
			return fmt.Errorf("insert metric '%s' failed: %w", name, err)
		}
	}
	return nil
}

func (s *Storage) updateDeviceStatus(ctx context.Context, tx pgx.Tx, data *validator.ValidatedTelemetry) error {
	query := `UPDATE devices 
	          SET status = 'online', 
	              last_seen = GREATEST(COALESCE(last_seen, '1970-01-01'::timestamptz), $1),
	              updated_at = NOW()
	          WHERE id = $2`

	_, err := tx.Exec(ctx, query, data.DeviceTime, data.DeviceID)
	return err
}

func (s *Storage) updateRedisCacheIfNewest(ctx context.Context, data *validator.ValidatedTelemetry) error {
	key := fmt.Sprintf(RedisDeviceLatestKeyFormat, data.DeviceID)
	existing, err := s.Redis.Get(ctx, key).Result()
	if err == nil {
		var cached CachedTelemetry
		if json.Unmarshal([]byte(existing), &cached) == nil && cached.Timestamp.After(data.DeviceTime) {
			return nil
		}
	}

	payload := CachedTelemetry{
		DeviceID:   data.DeviceID,
		Timestamp:  data.DeviceTime,
		ReceivedAt: data.ReceivedAt,
		Metrics:    data.Metrics,
		UpdatedAt:  time.Now().UTC(),
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	liveChannel := fmt.Sprintf("telemetry:device:%s", data.DeviceID)
	_ = s.Redis.Publish(ctx, liveChannel, encoded).Err()

	return s.Redis.Set(ctx, key, encoded, RedisLatestTTL).Err()
}

// EnsureSchema ensures hypertable, received_at column, retention policy, and continuous aggregates.
func (s *Storage) EnsureSchema(ctx context.Context) error {
	columnQuery := `DO $$
	BEGIN
	    IF NOT EXISTS (
	        SELECT 1 FROM information_schema.columns 
	        WHERE table_name = 'telemetry' AND column_name = 'received_at'
	    ) THEN
	        ALTER TABLE telemetry ADD COLUMN received_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
	    END IF;
	END $$;`
	if _, err := s.Pool.Exec(ctx, columnQuery); err != nil {
		return fmt.Errorf("failed ensuring received_at column: %w", err)
	}

	return s.ensureTimescalePolicies(ctx)
}

func (s *Storage) ensureTimescalePolicies(ctx context.Context) error {
	var hasTimescale bool
	_ = s.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").Scan(&hasTimescale)
	if !hasTimescale {
		return nil
	}

	policyQuery := `DO $$
	BEGIN
	    IF NOT EXISTS (
	        SELECT 1 FROM timescaledb_information.continuous_aggregates WHERE view_name = 'telemetry_hourly'
	    ) THEN
	        CREATE MATERIALIZED VIEW telemetry_hourly
	        WITH (timescaledb.continuous) AS
	        SELECT
	            time_bucket('1 hour', time) AS bucket,
	            device_id,
	            metric_name,
	            AVG(metric_value) AS avg_value,
	            MIN(metric_value) AS min_value,
	            MAX(metric_value) AS max_value,
	            COUNT(metric_value) AS count_value
	        FROM telemetry
	        GROUP BY bucket, device_id, metric_name
	        WITH NO DATA;
	    END IF;

	    PERFORM add_continuous_aggregate_policy('telemetry_hourly',
	        start_offset => INTERVAL '3 days',
	        end_offset => INTERVAL '1 hour',
	        schedule_interval => INTERVAL '1 hour',
	        if_not_exists => TRUE);

	    PERFORM add_retention_policy('telemetry', INTERVAL '30 days', if_not_exists => TRUE);
	END $$;`

	_, err := s.Pool.Exec(ctx, policyQuery)
	return err
}

func initDatabasePool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid db dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to postgres: %w", err)
	}

	return pool, nil
}

// GetTelemetryPoints retrieves points for a device and metric ordered by time descending.
func (s *Storage) GetTelemetryPoints(ctx context.Context, deviceID, metricName string, limit int) ([]TelemetryPoint, error) {
	if limit <= 0 {
		limit = DefaultQueryLimit
	}

	query := `SELECT time, received_at, device_id, metric_name, metric_value 
	          FROM telemetry 
	          WHERE device_id = $1 AND metric_name = $2 
	          ORDER BY time DESC LIMIT $3`

	rows, err := s.Pool.Query(ctx, query, deviceID, metricName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TelemetryPoint
	for rows.Next() {
		var p TelemetryPoint
		if err := rows.Scan(&p.Time, &p.ReceivedAt, &p.DeviceID, &p.MetricName, &p.MetricValue); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

// GetHourlyAggregates queries the continuous aggregate view for hourly downsampled metrics.
func (s *Storage) GetHourlyAggregates(ctx context.Context, deviceID, metricName string, limit int) ([]HourlyAggregate, error) {
	if limit <= 0 {
		limit = DefaultQueryLimit
	}

	query := `SELECT bucket, device_id, metric_name, avg_value, min_value, max_value, count_value 
	          FROM telemetry_hourly 
	          WHERE device_id = $1 AND metric_name = $2 
	          ORDER BY bucket DESC LIMIT $3`

	rows, err := s.Pool.Query(ctx, query, deviceID, metricName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aggs []HourlyAggregate
	for rows.Next() {
		var a HourlyAggregate
		if err := rows.Scan(&a.Bucket, &a.DeviceID, &a.MetricName, &a.AvgValue, &a.MinValue, &a.MaxValue, &a.CountValue); err != nil {
			return nil, err
		}
		aggs = append(aggs, a)
	}
	return aggs, nil
}

// CheckRetentionPolicy checks if a retention policy exists on the given hypertable.
func (s *Storage) CheckRetentionPolicy(ctx context.Context, hypertableName string) (bool, string, error) {
	query := `SELECT config->>'drop_after' 
	          FROM timescaledb_information.jobs 
	          WHERE application_name LIKE 'Retention Policy%'`

	var dropAfter string
	err := s.Pool.QueryRow(ctx, query).Scan(&dropAfter)
	if err != nil {
		return false, "", err
	}
	return true, dropAfter, nil
}

// CheckContinuousAggregate checks if continuous aggregate exists in TimescaleDB.
func (s *Storage) CheckContinuousAggregate(ctx context.Context, viewName string) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM timescaledb_information.continuous_aggregates 
		WHERE view_name = $1
	)`
	var exists bool
	err := s.Pool.QueryRow(ctx, query, viewName).Scan(&exists)
	return exists, err
}

// GetDeviceLastSeen returns last_seen timestamp for the device from devices table.
func (s *Storage) GetDeviceLastSeen(ctx context.Context, deviceID string) (*time.Time, error) {
	var lastSeen *time.Time
	err := s.Pool.QueryRow(ctx, "SELECT last_seen FROM devices WHERE id = $1", deviceID).Scan(&lastSeen)
	return lastSeen, err
}

// GetCachedLatestTelemetry retrieves latest cached telemetry from Redis.
func (s *Storage) GetCachedLatestTelemetry(ctx context.Context, deviceID string) (*CachedTelemetry, error) {
	key := fmt.Sprintf(RedisDeviceLatestKeyFormat, deviceID)
	val, err := s.Redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var cached CachedTelemetry
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

type IngestorRule struct {
	ID                  string
	Name                string
	Condition           string
	Threshold           float64
	Severity            string
	CooldownSeconds     int
	NotificationChannel string
	WebhookURL          string
}

func (s *Storage) evaluateRules(ctx context.Context, data *validator.ValidatedTelemetry) error {
	var orgID, deviceType string
	err := s.Pool.QueryRow(ctx, "SELECT organization_id, COALESCE(device_type, '') FROM devices WHERE id = $1", data.DeviceID).Scan(&orgID, &deviceType)
	if err != nil {
		return nil
	}

	for metricName, metricValue := range data.Metrics {
		s.evaluateMetricRules(ctx, orgID, deviceType, data.DeviceID, metricName, metricValue, data.DeviceTime)
	}
	return nil
}

func (s *Storage) evaluateMetricRules(ctx context.Context, orgID, deviceType, deviceID, metricName string, metricValue float64, triggeredAt time.Time) {
	query := `SELECT id, name, condition, threshold, severity, cooldown_seconds, notification_channel, webhook_url
	          FROM rules
	          WHERE organization_id = $1 AND enabled = TRUE AND metric_name = $2
	            AND (device_id IS NULL OR device_id = $3)
	            AND (device_type = '' OR device_type = $4)`

	rows, err := s.Pool.Query(ctx, query, orgID, metricName, deviceID, deviceType)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var r IngestorRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Condition, &r.Threshold, &r.Severity, &r.CooldownSeconds, &r.NotificationChannel, &r.WebhookURL); err != nil {
			continue
		}

		if !evaluateCondition(r.Condition, metricValue, r.Threshold) {
			continue
		}

		cooldownKey := fmt.Sprintf("cooldown:rule:%s:device:%s", r.ID, deviceID)
		if s.Redis != nil {
			exists, _ := s.Redis.Exists(ctx, cooldownKey).Result()
			if exists > 0 {
				continue
			}
			_ = s.Redis.Set(ctx, cooldownKey, "1", time.Duration(r.CooldownSeconds)*time.Second).Err()
		}

		s.triggerIngestorAlert(ctx, orgID, deviceID, metricName, metricValue, &r, triggeredAt)
	}
}

func (s *Storage) triggerIngestorAlert(ctx context.Context, orgID, deviceID, metricName string, metricValue float64, r *IngestorRule, triggeredAt time.Time) {
	shortDev := deviceID
	if len(shortDev) > 6 {
		shortDev = shortDev[:6]
	}
	alertID := fmt.Sprintf("alt_%d_%s", time.Now().UnixNano(), shortDev)
	notificationID := fmt.Sprintf("ntf_%d", time.Now().UnixNano())
	msg := fmt.Sprintf("%s (%.2f %s %.2f) detected on device %s", r.Name, metricValue, r.Condition, r.Threshold, deviceID)

	alertQuery := `INSERT INTO alerts (id, organization_id, rule_id, device_id, severity, status, message, metric_name, metric_value, threshold_value, triggered_at, created_at, updated_at)
	               VALUES ($1, $2, $3, $4, $5, 'active', $6, $7, $8, $9, $10, NOW(), NOW())`
	_, _ = s.Pool.Exec(ctx, alertQuery, alertID, orgID, r.ID, deviceID, r.Severity, msg, metricName, metricValue, r.Threshold, triggeredAt)

	notifQuery := `INSERT INTO notifications (id, organization_id, alert_id, title, message, severity, channel, read, created_at)
	               VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE, NOW())`
	title := fmt.Sprintf("[%s] %s", strings.ToUpper(r.Severity), r.Name)
	_, _ = s.Pool.Exec(ctx, notifQuery, notificationID, orgID, alertID, title, msg, r.Severity, r.NotificationChannel)

	if s.Redis != nil {
		eventPayload := map[string]interface{}{
			"alert": map[string]interface{}{
				"id": alertID, "organization_id": orgID, "rule_id": r.ID, "device_id": deviceID,
				"severity": r.Severity, "status": "active", "message": msg,
				"metric_name": metricName, "metric_value": metricValue, "threshold_value": r.Threshold,
				"triggered_at": triggeredAt,
			},
			"notification": map[string]interface{}{
				"id": notificationID, "organization_id": orgID, "alert_id": alertID,
				"title": title, "message": msg, "severity": r.Severity,
				"channel": r.NotificationChannel, "read": false, "created_at": time.Now().UTC(),
			},
		}
		if encoded, err := json.Marshal(eventPayload); err == nil {
			channel := fmt.Sprintf("org:%s:notifications", orgID)
			_ = s.Redis.Publish(ctx, channel, encoded).Err()
		}
	}
}

func evaluateCondition(condition string, actualValue, threshold float64) bool {
	switch condition {
	case ">":
		return actualValue > threshold
	case ">=":
		return actualValue >= threshold
	case "<":
		return actualValue < threshold
	case "<=":
		return actualValue <= threshold
	case "==":
		return math.Abs(actualValue-threshold) < 1e-9
	case "!=":
		return math.Abs(actualValue-threshold) >= 1e-9
	default:
		return false
	}
}

