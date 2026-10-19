DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
ALTER TABLE devices DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS organizations;
