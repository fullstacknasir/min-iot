-- ==========================================================
-- minIoT TimescaleDB / PostgreSQL Initialization Script
-- ==========================================================

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    device_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'offline',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Device Telemetry Hypertable
CREATE TABLE IF NOT EXISTS telemetry (
    time TIMESTAMPTZ NOT NULL,
    device_id VARCHAR(64) NOT NULL,
    metric_name VARCHAR(64) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    string_value TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (time, device_id, metric_name)
);

-- Convert telemetry table into a TimescaleDB hypertable
SELECT create_hypertable('telemetry', 'time', if_not_exists => TRUE);

-- Hourly continuous aggregate view
CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry_hourly
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

-- Continuous aggregate policy (refreshing every hour)
SELECT add_continuous_aggregate_policy('telemetry_hourly',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

-- 30-day raw data retention policy
SELECT add_retention_policy('telemetry', INTERVAL '30 days', if_not_exists => TRUE);

-- Indexes for efficient time-series querying
CREATE INDEX IF NOT EXISTS idx_telemetry_device_time ON telemetry (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices (status);
CREATE INDEX IF NOT EXISTS idx_devices_type ON devices (device_type);

-- Seed initial test devices
INSERT INTO devices (id, name, device_type, status, metadata)
VALUES 
    ('dev-sensor-001', 'Environmental Sensor Alpha', 'temperature_humidity', 'offline', '{"location": "Warehouse A", "floor": 2, "interval_sec": 3}'),
    ('dev-sensor-002', 'Power Meter Beta', 'power_monitor', 'offline', '{"location": "Server Room", "rack": 4, "interval_sec": 3}'),
    ('dev-sensor-003', 'HVAC Controller Gamma', 'hvac', 'offline', '{"location": "Building 1", "zone": "North", "interval_sec": 5}')
ON CONFLICT (id) DO NOTHING;

-- Seed sample telemetry data for quick visualization
INSERT INTO telemetry (time, device_id, metric_name, metric_value)
VALUES
    (NOW() - INTERVAL '10 minutes', 'dev-sensor-001', 'temperature', 22.4),
    (NOW() - INTERVAL '10 minutes', 'dev-sensor-001', 'humidity', 54.2),
    (NOW() - INTERVAL '5 minutes', 'dev-sensor-001', 'temperature', 23.1),
    (NOW() - INTERVAL '5 minutes', 'dev-sensor-001', 'humidity', 53.8),
    (NOW(), 'dev-sensor-001', 'temperature', 23.8),
    (NOW(), 'dev-sensor-001', 'humidity', 52.9),
    (NOW() - INTERVAL '5 minutes', 'dev-sensor-002', 'voltage', 229.8),
    (NOW() - INTERVAL '5 minutes', 'dev-sensor-002', 'current', 4.3),
    (NOW(), 'dev-sensor-002', 'voltage', 230.1),
    (NOW(), 'dev-sensor-002', 'current', 4.5)
ON CONFLICT DO NOTHING;
