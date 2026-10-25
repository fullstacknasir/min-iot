package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MigrationTimeout = 15 * time.Second
)

type MigrationItem struct {
	Version string
	SQL     string
}

var allMigrations = []MigrationItem{
	{
		Version: "000001_auth_schema",
		SQL: `
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'devices' AND column_name = 'organization_id'
    ) THEN
        ALTER TABLE devices ADD COLUMN organization_id VARCHAR(64) REFERENCES organizations(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_users_org_id ON users (organization_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_devices_org_id ON devices (organization_id);
`,
	},
	{
		Version: "000002_device_management",
		SQL: `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'devices' AND column_name = 'last_seen'
    ) THEN
        ALTER TABLE devices ADD COLUMN last_seen TIMESTAMPTZ;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS device_credentials (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    credential_type VARCHAR(32) NOT NULL DEFAULT 'token',
    secret_hash VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'revoked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_device_credentials_device_id ON device_credentials (device_id);
CREATE INDEX IF NOT EXISTS idx_device_credentials_org_id ON device_credentials (organization_id);
CREATE INDEX IF NOT EXISTS idx_device_credentials_hash ON device_credentials (secret_hash);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices (last_seen);
`,
	},
	{
		Version: "000003_telemetry_hypertable_aggregates",
		SQL: `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'telemetry' AND column_name = 'received_at'
    ) THEN
        ALTER TABLE telemetry ADD COLUMN received_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
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
    END IF;
END $$;
`,
	},
	{
		Version: "000004_device_assignment_and_user_profile",
		SQL: `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'devices' AND column_name = 'assigned_user_id'
    ) THEN
        ALTER TABLE devices ADD COLUMN assigned_user_id VARCHAR(64) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'name'
    ) THEN
        ALTER TABLE users ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_devices_assigned_user_id ON devices (assigned_user_id);
`,
	},
	{
		Version: "000005_rules_alerts_notifications",
		SQL: `
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    device_id VARCHAR(64) REFERENCES devices(id) ON DELETE CASCADE,
    device_type VARCHAR(64) NOT NULL DEFAULT '',
    metric_name VARCHAR(64) NOT NULL,
    condition VARCHAR(8) NOT NULL CHECK (condition IN ('>', '>=', '<', '<=', '==', '!=')),
    threshold DOUBLE PRECISION NOT NULL,
    severity VARCHAR(32) NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical', 'warning', 'info')),
    cooldown_seconds INT NOT NULL DEFAULT 60,
    notification_channel VARCHAR(32) NOT NULL DEFAULT 'in_app' CHECK (notification_channel IN ('in_app', 'email', 'webhook', 'all')),
    webhook_url TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rules_org_enabled ON rules (organization_id, enabled);
CREATE INDEX IF NOT EXISTS idx_rules_metric ON rules (metric_name);
CREATE INDEX IF NOT EXISTS idx_rules_device ON rules (device_id);

CREATE TABLE IF NOT EXISTS alerts (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    rule_id VARCHAR(64) REFERENCES rules(id) ON DELETE SET NULL,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    severity VARCHAR(32) NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
    message TEXT NOT NULL,
    metric_name VARCHAR(64) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    threshold_value DOUBLE PRECISION NOT NULL,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(64) REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(64) REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_org_status ON alerts (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_alerts_device ON alerts (device_id);
CREATE INDEX IF NOT EXISTS idx_alerts_rule ON alerts (rule_id);
CREATE INDEX IF NOT EXISTS idx_alerts_triggered_at ON alerts (triggered_at DESC);

CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    alert_id VARCHAR(64) REFERENCES alerts(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(32) NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical', 'warning', 'info')),
    channel VARCHAR(32) NOT NULL DEFAULT 'in_app' CHECK (channel IN ('in_app', 'email', 'webhook')),
    read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_org_read ON notifications (organization_id, read);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications (created_at DESC);
`,
	},
	{
		Version: "000006_dashboards",
		SQL: `
CREATE TABLE IF NOT EXISTS dashboards (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    configuration JSONB NOT NULL DEFAULT '{"widgets": []}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboards_org ON dashboards (organization_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_default ON dashboards (organization_id, is_default);
`,
	},
	{
		Version: "000007_custom_widgets",
		SQL: `
CREATE TABLE IF NOT EXISTS custom_widgets (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    widget_type VARCHAR(64) NOT NULL DEFAULT 'telemetry',
    template_html TEXT NOT NULL DEFAULT '',
    style_css TEXT NOT NULL DEFAULT '',
    controller_js TEXT NOT NULL DEFAULT '',
    settings_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_widgets_org ON custom_widgets (organization_id);
CREATE INDEX IF NOT EXISTS idx_custom_widgets_updated ON custom_widgets (organization_id, updated_at DESC);
`,
	},
}

// RunMigrations applies all pending database schema migrations in order.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationCtx, cancel := context.WithTimeout(ctx, MigrationTimeout)
	defer cancel()

	if err := createMigrationsTable(migrationCtx, pool); err != nil {
		return fmt.Errorf("failed to create schema_migrations: %w", err)
	}

	for _, m := range allMigrations {
		applied, err := isMigrationApplied(migrationCtx, pool, m.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := applyMigration(migrationCtx, pool, m.Version, m.SQL); err != nil {
			return err
		}
	}
	return nil
}

func createMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`
	_, err := pool.Exec(ctx, query)
	return err
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`
	var exists bool
	err := pool.QueryRow(ctx, query, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed checking migration %s: %w", version, err)
	}
	return exists, nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version string, sql string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed executing migration %s: %w", version, err)
	}

	recordQuery := `INSERT INTO schema_migrations (version) VALUES ($1)`
	if _, err := tx.Exec(ctx, recordQuery, version); err != nil {
		return fmt.Errorf("failed recording migration %s: %w", version, err)
	}

	log.Printf("[Migration] Applied migration %s successfully", version)
	return tx.Commit(ctx)
}
