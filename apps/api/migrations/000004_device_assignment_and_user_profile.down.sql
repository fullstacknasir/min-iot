-- Rollback Migration 000004
DROP INDEX IF EXISTS idx_devices_assigned_user_id;

ALTER TABLE devices DROP COLUMN IF EXISTS assigned_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS name;
