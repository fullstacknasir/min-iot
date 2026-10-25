-- ==========================================================
-- Migration 000004: Device Assignment & User Profile
-- ==========================================================

-- Add assigned_user_id to devices table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'devices' AND column_name = 'assigned_user_id'
    ) THEN
        ALTER TABLE devices ADD COLUMN assigned_user_id VARCHAR(64) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Add name to users table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'name'
    ) THEN
        ALTER TABLE users ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '';
    END IF;
END $$;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_devices_assigned_user_id ON devices (assigned_user_id);
