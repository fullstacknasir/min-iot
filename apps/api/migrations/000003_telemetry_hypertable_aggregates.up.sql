-- Migration 000003: Telemetry Hypertable received_at, Continuous Aggregate, and Retention Policy

-- Ensure received_at column exists in telemetry table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'telemetry' AND column_name = 'received_at'
    ) THEN
        ALTER TABLE telemetry ADD COLUMN received_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
    END IF;
END $$;

-- Enable TimescaleDB continuous aggregate and retention policies if extension is present
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        -- Create hourly continuous aggregate materialized view if not already created
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

        -- Continuous aggregate refresh policy (1 hour schedule, refreshing last 3 days up to last 1 hour)
        PERFORM add_continuous_aggregate_policy('telemetry_hourly',
            start_offset => INTERVAL '3 days',
            end_offset => INTERVAL '1 hour',
            schedule_interval => INTERVAL '1 hour',
            if_not_exists => TRUE);

        -- 30-day raw data retention policy
        PERFORM add_retention_policy('telemetry', INTERVAL '30 days', if_not_exists => TRUE);
    END IF;
END $$;
