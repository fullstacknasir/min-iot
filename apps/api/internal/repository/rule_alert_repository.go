package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"miniot/apps/api/internal/models"
)

var (
	ErrRuleNotFound  = errors.New("rule not found")
	ErrAlertNotFound = errors.New("alert not found")
)

type RuleAlertRepository struct {
	pool *pgxpool.Pool
}

func NewRuleAlertRepository(pool *pgxpool.Pool) *RuleAlertRepository {
	return &RuleAlertRepository{pool: pool}
}

// ---------------------------------------------------------------------------
// Rules Operations
// ---------------------------------------------------------------------------

func (r *RuleAlertRepository) CreateRule(ctx context.Context, rule *models.Rule) error {
	query := `INSERT INTO rules (
		id, organization_id, name, description, device_id, device_type,
		metric_name, condition, threshold, severity, cooldown_seconds,
		notification_channel, webhook_url, enabled, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.OrganizationID, rule.Name, rule.Description, rule.DeviceID, rule.DeviceType,
		rule.MetricName, rule.Condition, rule.Threshold, rule.Severity, rule.CooldownSeconds,
		rule.NotificationChannel, rule.WebhookURL, rule.Enabled, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

func (r *RuleAlertRepository) GetRuleByID(ctx context.Context, id, orgID string) (*models.Rule, error) {
	query := `SELECT r.id, r.organization_id, r.name, r.description, r.device_id, d.name,
	                 r.device_type, r.metric_name, r.condition, r.threshold, r.severity,
	                 r.cooldown_seconds, r.notification_channel, r.webhook_url, r.enabled,
	                 r.created_at, r.updated_at
	          FROM rules r
	          LEFT JOIN devices d ON r.device_id = d.id
	          WHERE r.id = $1 AND r.organization_id = $2`

	var rule models.Rule
	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&rule.ID, &rule.OrganizationID, &rule.Name, &rule.Description, &rule.DeviceID, &rule.DeviceName,
		&rule.DeviceType, &rule.MetricName, &rule.Condition, &rule.Threshold, &rule.Severity,
		&rule.CooldownSeconds, &rule.NotificationChannel, &rule.WebhookURL, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRuleNotFound
	}
	return &rule, err
}

func (r *RuleAlertRepository) ListRules(ctx context.Context, orgID string, page, limit int, deviceID string, enabled *bool) ([]models.Rule, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	conditions = append(conditions, fmt.Sprintf("r.organization_id = $%d", idx))
	args = append(args, orgID)
	idx++

	if deviceID != "" {
		conditions = append(conditions, fmt.Sprintf("(r.device_id = $%d OR r.device_id IS NULL)", idx))
		args = append(args, deviceID)
		idx++
	}

	if enabled != nil {
		conditions = append(conditions, fmt.Sprintf("r.enabled = $%d", idx))
		args = append(args, *enabled)
		idx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM rules r WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := fmt.Sprintf(`SELECT r.id, r.organization_id, r.name, r.description, r.device_id, d.name,
	                                 r.device_type, r.metric_name, r.condition, r.threshold, r.severity,
	                                 r.cooldown_seconds, r.notification_channel, r.webhook_url, r.enabled,
	                                 r.created_at, r.updated_at
	                          FROM rules r
	                          LEFT JOIN devices d ON r.device_id = d.id
	                          WHERE %s
	                          ORDER BY r.created_at DESC`, whereClause)

	if limit > 0 {
		offset := (page - 1) * limit
		if offset < 0 {
			offset = 0
		}
		dataQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rules := make([]models.Rule, 0)
	for rows.Next() {
		var item models.Rule
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Name, &item.Description, &item.DeviceID, &item.DeviceName,
			&item.DeviceType, &item.MetricName, &item.Condition, &item.Threshold, &item.Severity,
			&item.CooldownSeconds, &item.NotificationChannel, &item.WebhookURL, &item.Enabled,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		rules = append(rules, item)
	}
	return rules, total, nil
}

func (r *RuleAlertRepository) UpdateRule(ctx context.Context, rule *models.Rule) error {
	query := `UPDATE rules SET
		name = $1, description = $2, device_id = $3, device_type = $4,
		metric_name = $5, condition = $6, threshold = $7, severity = $8,
		cooldown_seconds = $9, notification_channel = $10, webhook_url = $11,
		enabled = $12, updated_at = NOW()
		WHERE id = $13 AND organization_id = $14`

	cmd, err := r.pool.Exec(ctx, query,
		rule.Name, rule.Description, rule.DeviceID, rule.DeviceType,
		rule.MetricName, rule.Condition, rule.Threshold, rule.Severity,
		rule.CooldownSeconds, rule.NotificationChannel, rule.WebhookURL,
		rule.Enabled, rule.ID, rule.OrganizationID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (r *RuleAlertRepository) DeleteRule(ctx context.Context, id, orgID string) error {
	cmd, err := r.pool.Exec(ctx, "DELETE FROM rules WHERE id = $1 AND organization_id = $2", id, orgID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (r *RuleAlertRepository) ToggleRule(ctx context.Context, id, orgID string) (bool, error) {
	query := `UPDATE rules SET enabled = NOT enabled, updated_at = NOW()
	          WHERE id = $1 AND organization_id = $2
	          RETURNING enabled`
	var newStatus bool
	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(&newStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrRuleNotFound
	}
	return newStatus, err
}

func (r *RuleAlertRepository) FindActiveRulesForDevice(ctx context.Context, orgID, deviceID, deviceType, metricName string) ([]models.Rule, error) {
	query := `SELECT id, organization_id, name, description, device_id, device_type,
	                 metric_name, condition, threshold, severity, cooldown_seconds,
	                 notification_channel, webhook_url, enabled, created_at, updated_at
	          FROM rules
	          WHERE organization_id = $1
	            AND enabled = TRUE
	            AND metric_name = $2
	            AND (device_id IS NULL OR device_id = $3)
	            AND (device_type = '' OR device_type = $4)`

	rows, err := r.pool.Query(ctx, query, orgID, metricName, deviceID, deviceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.Rule
	for rows.Next() {
		var item models.Rule
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Name, &item.Description, &item.DeviceID, &item.DeviceType,
			&item.MetricName, &item.Condition, &item.Threshold, &item.Severity, &item.CooldownSeconds,
			&item.NotificationChannel, &item.WebhookURL, &item.Enabled, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, item)
	}
	return rules, nil
}

// ---------------------------------------------------------------------------
// Alerts Operations
// ---------------------------------------------------------------------------

func (r *RuleAlertRepository) CreateAlert(ctx context.Context, alert *models.Alert) error {
	query := `INSERT INTO alerts (
		id, organization_id, rule_id, device_id, severity, status, message,
		metric_name, metric_value, threshold_value, triggered_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	now := time.Now().UTC()
	alert.CreatedAt = now
	alert.UpdatedAt = now
	if alert.TriggeredAt.IsZero() {
		alert.TriggeredAt = now
	}

	_, err := r.pool.Exec(ctx, query,
		alert.ID, alert.OrganizationID, alert.RuleID, alert.DeviceID, alert.Severity,
		alert.Status, alert.Message, alert.MetricName, alert.MetricValue, alert.ThresholdValue,
		alert.TriggeredAt, alert.CreatedAt, alert.UpdatedAt,
	)
	return err
}

func (r *RuleAlertRepository) GetAlertByID(ctx context.Context, id, orgID string) (*models.Alert, error) {
	query := `SELECT a.id, a.organization_id, a.rule_id, r.name, a.device_id, d.name,
	                 a.severity, a.status, a.message, a.metric_name, a.metric_value,
	                 a.threshold_value, a.triggered_at, a.acknowledged_at, a.acknowledged_by,
	                 a.resolved_at, a.resolved_by, a.created_at, a.updated_at
	          FROM alerts a
	          LEFT JOIN rules r ON a.rule_id = r.id
	          LEFT JOIN devices d ON a.device_id = d.id
	          WHERE a.id = $1 AND a.organization_id = $2`

	var alert models.Alert
	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&alert.ID, &alert.OrganizationID, &alert.RuleID, &alert.RuleName, &alert.DeviceID, &alert.DeviceName,
		&alert.Severity, &alert.Status, &alert.Message, &alert.MetricName, &alert.MetricValue,
		&alert.ThresholdValue, &alert.TriggeredAt, &alert.AcknowledgedAt, &alert.AcknowledgedBy,
		&alert.ResolvedAt, &alert.ResolvedBy, &alert.CreatedAt, &alert.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAlertNotFound
	}
	return &alert, err
}

func (r *RuleAlertRepository) ListAlerts(ctx context.Context, orgID string, page, limit int, status, severity, deviceID string) ([]models.Alert, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	conditions = append(conditions, fmt.Sprintf("a.organization_id = $%d", idx))
	args = append(args, orgID)
	idx++

	if status != "" && status != "all" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", idx))
		args = append(args, status)
		idx++
	}

	if severity != "" && severity != "all" {
		conditions = append(conditions, fmt.Sprintf("a.severity = $%d", idx))
		args = append(args, severity)
		idx++
	}

	if deviceID != "" {
		conditions = append(conditions, fmt.Sprintf("a.device_id = $%d", idx))
		args = append(args, deviceID)
		idx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alerts a WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := fmt.Sprintf(`SELECT a.id, a.organization_id, a.rule_id, r.name, a.device_id, d.name,
	                                 a.severity, a.status, a.message, a.metric_name, a.metric_value,
	                                 a.threshold_value, a.triggered_at, a.acknowledged_at, a.acknowledged_by,
	                                 a.resolved_at, a.resolved_by, a.created_at, a.updated_at
	                          FROM alerts a
	                          LEFT JOIN rules r ON a.rule_id = r.id
	                          LEFT JOIN devices d ON a.device_id = d.id
	                          WHERE %s
	                          ORDER BY a.triggered_at DESC`, whereClause)

	if limit > 0 {
		offset := (page - 1) * limit
		if offset < 0 {
			offset = 0
		}
		dataQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var item models.Alert
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.RuleID, &item.RuleName, &item.DeviceID, &item.DeviceName,
			&item.Severity, &item.Status, &item.Message, &item.MetricName, &item.MetricValue,
			&item.ThresholdValue, &item.TriggeredAt, &item.AcknowledgedAt, &item.AcknowledgedBy,
			&item.ResolvedAt, &item.ResolvedBy, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, item)
	}
	return alerts, total, nil
}

func (r *RuleAlertRepository) AcknowledgeAlert(ctx context.Context, id, orgID, userID string) error {
	query := `UPDATE alerts 
	          SET status = 'acknowledged', acknowledged_at = NOW(), acknowledged_by = $1, updated_at = NOW()
	          WHERE id = $2 AND organization_id = $3 AND status = 'active'`
	cmd, err := r.pool.Exec(ctx, query, userID, id, orgID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

func (r *RuleAlertRepository) ResolveAlert(ctx context.Context, id, orgID, userID string) error {
	query := `UPDATE alerts 
	          SET status = 'resolved', resolved_at = NOW(), resolved_by = $1, updated_at = NOW()
	          WHERE id = $2 AND organization_id = $3 AND status != 'resolved'`
	cmd, err := r.pool.Exec(ctx, query, userID, id, orgID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Notifications Operations
// ---------------------------------------------------------------------------

func (r *RuleAlertRepository) CreateNotification(ctx context.Context, n *models.Notification) error {
	query := `INSERT INTO notifications (id, organization_id, alert_id, title, message, severity, channel, read, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}

	_, err := r.pool.Exec(ctx, query,
		n.ID, n.OrganizationID, n.AlertID, n.Title, n.Message, n.Severity, n.Channel, n.Read, n.CreatedAt,
	)
	return err
}

func (r *RuleAlertRepository) ListNotifications(ctx context.Context, orgID string, page, limit int, unreadOnly bool) ([]models.Notification, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	conditions = append(conditions, fmt.Sprintf("organization_id = $%d", idx))
	args = append(args, orgID)
	idx++

	if unreadOnly {
		conditions = append(conditions, fmt.Sprintf("read = $%d", idx))
		args = append(args, false)
		idx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := fmt.Sprintf(`SELECT id, organization_id, alert_id, title, message, severity, channel, read, created_at
	                          FROM notifications
	                          WHERE %s
	                          ORDER BY created_at DESC`, whereClause)

	if limit > 0 {
		offset := (page - 1) * limit
		if offset < 0 {
			offset = 0
		}
		dataQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	notifications := make([]models.Notification, 0)
	for rows.Next() {
		var item models.Notification
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.AlertID, &item.Title, &item.Message,
			&item.Severity, &item.Channel, &item.Read, &item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, item)
	}
	return notifications, total, nil
}

func (r *RuleAlertRepository) GetUnreadNotificationCount(ctx context.Context, orgID string) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE organization_id = $1 AND read = FALSE`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	return count, err
}

func (r *RuleAlertRepository) MarkNotificationAsRead(ctx context.Context, id, orgID string) error {
	query := `UPDATE notifications SET read = TRUE WHERE id = $1 AND organization_id = $2`
	_, err := r.pool.Exec(ctx, query, id, orgID)
	return err
}

func (r *RuleAlertRepository) MarkAllNotificationsAsRead(ctx context.Context, orgID string) error {
	query := `UPDATE notifications SET read = TRUE WHERE organization_id = $1 AND read = FALSE`
	_, err := r.pool.Exec(ctx, query, orgID)
	return err
}
