-- Migration: 000005_rules_alerts_notifications.up.sql
-- Rule Engine, Alert Management, and Multi-channel Notifications

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
