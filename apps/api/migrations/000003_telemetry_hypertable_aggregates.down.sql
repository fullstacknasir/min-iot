-- Migration 000003: Revert Telemetry Hypertable received_at, Continuous Aggregate, and Retention Policy

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM remove_retention_policy('telemetry', if_exists => TRUE);
        PERFORM remove_continuous_aggregate_policy('telemetry_hourly', if_exists => TRUE);
        DROP MATERIALIZED VIEW IF EXISTS telemetry_hourly CASCADE;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'telemetry' AND column_name = 'received_at'
    ) THEN
        ALTER TABLE telemetry DROP COLUMN received_at;
    END IF;
END $$;
