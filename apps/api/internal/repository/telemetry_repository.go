package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"miniot/apps/api/internal/models"
)

const (
	RedisDeviceLatestKeyPrefix = "device:%s:latest"
	RedisLiveTelemetryChannel  = "telemetry:device:%s"
	DefaultHistoryLimit        = 100
	MaxHistoryLimit            = 1000
	DefaultAggregateBucket     = "1 hour"
	RedisLatestTTL             = 24 * time.Hour
)

var (
	ErrDeviceNotFoundInOrg = errors.New("device not found in organization")
	ErrInvalidTimeRange    = errors.New("invalid time range: 'from' must be before 'to'")
	ErrInvalidBucketFormat = errors.New("invalid bucket format; e.g. '15 minutes', '1 hour', '1 day'")
)

type TelemetryRepository struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewTelemetryRepository(pool *pgxpool.Pool, rdb *redis.Client) *TelemetryRepository {
	return &TelemetryRepository{pool: pool, redis: rdb}
}

// GetLatestTelemetry retrieves cached telemetry from Redis with TimescaleDB fallback after verifying tenancy.
func (r *TelemetryRepository) GetLatestTelemetry(ctx context.Context, deviceID, organizationID string) (map[string]interface{}, error) {
	if err := r.verifyDeviceBelongsToOrg(ctx, deviceID, organizationID); err != nil {
		return nil, err
	}

	key := fmt.Sprintf(RedisDeviceLatestKeyPrefix, deviceID)
	data, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var result map[string]interface{}
		if json.Unmarshal([]byte(data), &result) == nil {
			return result, nil
		}
	}

	return r.fetchLatestFromHypertable(ctx, deviceID)
}

// GetTimeRangeHistory queries TimescaleDB hypertable for time-series points filtered by time range and metric.
func (r *TelemetryRepository) GetTimeRangeHistory(ctx context.Context, deviceID, organizationID, metricName string, from, to *time.Time, limit int, order string) ([]models.TelemetryDataPoint, error) {
	if err := r.verifyDeviceBelongsToOrg(ctx, deviceID, organizationID); err != nil {
		return nil, err
	}

	limit = clampLimit(limit)
	sortOrder := sanitizeSortOrder(order)

	query := fmt.Sprintf(`SELECT time, received_at, device_id, metric_name, metric_value, string_value 
	          FROM telemetry 
	          WHERE device_id = $1 
	            AND ($2::timestamptz IS NULL OR time >= $2)
	            AND ($3::timestamptz IS NULL OR time <= $3)
	            AND ($4 = '' OR metric_name = $4)
	          ORDER BY time %s LIMIT $5`, sortOrder)

	rows, err := r.pool.Query(ctx, query, deviceID, from, to, metricName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query telemetry: %w", err)
	}
	defer rows.Close()

	return scanTelemetryPoints(rows)
}

// GetAggregates calculates avg, min, max, and count across time buckets using TimescaleDB time_bucket.
func (r *TelemetryRepository) GetAggregates(ctx context.Context, deviceID, organizationID, bucket, metricName string, from, to *time.Time, limit int) ([]models.TelemetryAggregatePoint, error) {
	if err := r.verifyDeviceBelongsToOrg(ctx, deviceID, organizationID); err != nil {
		return nil, err
	}

	validBucket := sanitizeBucket(bucket)
	limit = clampLimit(limit)

	query := `SELECT 
	              time_bucket($1::interval, time) AS bucket,
	              device_id,
	              metric_name,
	              COALESCE(AVG(metric_value), 0) AS avg_value,
	              COALESCE(MIN(metric_value), 0) AS min_value,
	              COALESCE(MAX(metric_value), 0) AS max_value,
	              COUNT(metric_value) AS count_value
	          FROM telemetry
	          WHERE device_id = $2
	            AND ($3::timestamptz IS NULL OR time >= $3)
	            AND ($4::timestamptz IS NULL OR time <= $4)
	            AND ($5 = '' OR metric_name = $5)
	          GROUP BY bucket, device_id, metric_name
	          ORDER BY bucket DESC
	          LIMIT $6`

	rows, err := r.pool.Query(ctx, query, validBucket, deviceID, from, to, metricName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed querying aggregates: %w", err)
	}
	defer rows.Close()

	return scanAggregatePoints(rows)
}

// SubscribeLiveTelemetry creates a Redis Pub/Sub subscription for live device telemetry events.
func (r *TelemetryRepository) SubscribeLiveTelemetry(ctx context.Context, deviceID string) *redis.PubSub {
	channel := fmt.Sprintf(RedisLiveTelemetryChannel, deviceID)
	return r.redis.Subscribe(ctx, channel)
}

// VerifyDeviceBelongsToOrg asserts that the given device is owned by the target organization.
func (r *TelemetryRepository) VerifyDeviceBelongsToOrg(ctx context.Context, deviceID, organizationID string) error {
	return r.verifyDeviceBelongsToOrg(ctx, deviceID, organizationID)
}

// SaveBatch writes telemetry data points into TimescaleDB, updates latest Redis cache, and publishes live event.
func (r *TelemetryRepository) SaveBatch(ctx context.Context, deviceID, organizationID string, recordedAt time.Time, metrics map[string]float64) error {
	if organizationID != "" {
		if err := r.verifyDeviceBelongsToOrg(ctx, deviceID, organizationID); err != nil {
			return err
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	insertQuery := `INSERT INTO telemetry (time, received_at, device_id, metric_name, metric_value) 
	                VALUES ($1, $2, $3, $4, $5) ON CONFLICT (time, device_id, metric_name) DO UPDATE 
	                SET metric_value = EXCLUDED.metric_value, received_at = LEAST(telemetry.received_at, EXCLUDED.received_at)`

	now := time.Now().UTC()
	for metric, value := range metrics {
		if _, err := tx.Exec(ctx, insertQuery, recordedAt, now, deviceID, metric, value); err != nil {
			return fmt.Errorf("failed inserting metric %s: %w", metric, err)
		}
	}

	updateDeviceQuery := `UPDATE devices SET status = 'online', last_seen = GREATEST(COALESCE(last_seen, '1970-01-01'::timestamptz), $1), updated_at = NOW() WHERE id = $2`
	if _, err := tx.Exec(ctx, updateDeviceQuery, recordedAt, deviceID); err != nil {
		return fmt.Errorf("failed updating device status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return r.cacheAndPublishLiveTelemetry(ctx, deviceID, recordedAt, now, metrics)
}

func (r *TelemetryRepository) cacheAndPublishLiveTelemetry(ctx context.Context, deviceID string, timestamp, receivedAt time.Time, metrics map[string]float64) error {
	cachePayload := map[string]interface{}{
		"device_id":   deviceID,
		"timestamp":   timestamp,
		"received_at": receivedAt,
		"metrics":     metrics,
		"updated_at":  time.Now().UTC(),
	}
	cachedJSON, err := json.Marshal(cachePayload)
	if err != nil {
		return err
	}

	r.redis.Set(ctx, fmt.Sprintf(RedisDeviceLatestKeyPrefix, deviceID), cachedJSON, RedisLatestTTL)
	r.redis.Publish(ctx, fmt.Sprintf(RedisLiveTelemetryChannel, deviceID), cachedJSON)
	return nil
}

func (r *TelemetryRepository) fetchLatestFromHypertable(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	query := `SELECT DISTINCT ON (metric_name) time, received_at, metric_name, metric_value 
	          FROM telemetry 
	          WHERE device_id = $1 
	          ORDER BY metric_name, time DESC`

	rows, err := r.pool.Query(ctx, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("hypertable fallback query failed: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]float64)
	var latestTime time.Time
	var latestReceived time.Time

	for rows.Next() {
		var t, rec time.Time
		var name string
		var val float64
		if err := rows.Scan(&t, &rec, &name, &val); err == nil {
			metrics[name] = val
			if t.After(latestTime) {
				latestTime = t
				latestReceived = rec
			}
		}
	}

	if len(metrics) == 0 {
		return map[string]interface{}{"device_id": deviceID, "status": "no_data"}, nil
	}

	return map[string]interface{}{
		"device_id":   deviceID,
		"timestamp":   latestTime,
		"received_at": latestReceived,
		"metrics":     metrics,
	}, nil
}

func (r *TelemetryRepository) verifyDeviceBelongsToOrg(ctx context.Context, deviceID, organizationID string) error {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1 AND organization_id = $2)`
	err := r.pool.QueryRow(ctx, query, deviceID, organizationID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed verifying device ownership: %w", err)
	}
	if !exists {
		return ErrDeviceNotFoundInOrg
	}
	return nil
}

func scanTelemetryPoints(rows any) ([]models.TelemetryDataPoint, error) {
	pgxRows, ok := rows.(interface {
		Next() bool
		Scan(dest ...any) error
	})
	if !ok {
		return nil, errors.New("invalid rows interface")
	}

	points := make([]models.TelemetryDataPoint, 0)
	for pgxRows.Next() {
		var p models.TelemetryDataPoint
		if err := pgxRows.Scan(&p.Time, &p.ReceivedAt, &p.DeviceID, &p.MetricName, &p.MetricValue, &p.StringValue); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

func scanAggregatePoints(rows any) ([]models.TelemetryAggregatePoint, error) {
	pgxRows, ok := rows.(interface {
		Next() bool
		Scan(dest ...any) error
	})
	if !ok {
		return nil, errors.New("invalid rows interface")
	}

	aggs := make([]models.TelemetryAggregatePoint, 0)
	for pgxRows.Next() {
		var a models.TelemetryAggregatePoint
		if err := pgxRows.Scan(&a.Bucket, &a.DeviceID, &a.MetricName, &a.AvgValue, &a.MinValue, &a.MaxValue, &a.CountValue); err != nil {
			return nil, err
		}
		aggs = append(aggs, a)
	}
	return aggs, nil
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return DefaultHistoryLimit
	}
	if limit > MaxHistoryLimit {
		return MaxHistoryLimit
	}
	return limit
}

func sanitizeSortOrder(order string) string {
	if strings.ToUpper(strings.TrimSpace(order)) == "ASC" {
		return "ASC"
	}
	return "DESC"
}

var validBucketRegex = regexp.MustCompile(`^[0-9]+\s*[a-zA-Z]+$`)

func sanitizeBucket(bucket string) string {
	b := strings.TrimSpace(strings.ToLower(bucket))
	if b == "" || !validBucketRegex.MatchString(b) {
		return DefaultAggregateBucket
	}

	// Shorthand conversions
	switch {
	case strings.HasSuffix(b, "m") && !strings.HasSuffix(b, "m") == false && !strings.Contains(b, "min"):
		num := strings.TrimSuffix(b, "m")
		return num + " minutes"
	case strings.HasSuffix(b, "h") && !strings.Contains(b, "hour"):
		num := strings.TrimSuffix(b, "h")
		return num + " hours"
	case strings.HasSuffix(b, "d") && !strings.Contains(b, "day"):
		num := strings.TrimSuffix(b, "d")
		return num + " days"
	case strings.HasSuffix(b, "s") && !strings.Contains(b, "sec"):
		num := strings.TrimSuffix(b, "s")
		return num + " seconds"
	}

	allowedUnits := []string{"second", "seconds", "minute", "minutes", "hour", "hours", "day", "days", "week", "weeks"}
	for _, unit := range allowedUnits {
		if strings.HasSuffix(b, unit) {
			return b
		}
	}
	return DefaultAggregateBucket
}

