-- ==========================================================
-- Migration 000002: Device Management & Per-Device Credentials
-- ==========================================================

-- Add last_seen to devices if not present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'devices' AND column_name = 'last_seen'
    ) THEN
        ALTER TABLE devices ADD COLUMN last_seen TIMESTAMPTZ;
    END IF;
END $$;

-- Device Credentials Table (stores hashed credentials, raw secret shown once upon provisioning)
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

-- Indexes for fast multi-tenant lookup and authentication
CREATE INDEX IF NOT EXISTS idx_device_credentials_device_id ON device_credentials (device_id);
CREATE INDEX IF NOT EXISTS idx_device_credentials_org_id ON device_credentials (organization_id);
CREATE INDEX IF NOT EXISTS idx_device_credentials_hash ON device_credentials (secret_hash);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices (last_seen);
