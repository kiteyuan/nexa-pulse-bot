-- Drop ntfy delivery tables, settings, and send queue column.
DROP TABLE IF EXISTS ntfy_routes;
DROP TABLE IF EXISTS topics;

ALTER TABLE app_settings DROP COLUMN IF EXISTS ntfy_enabled;
ALTER TABLE app_settings DROP COLUMN IF EXISTS ntfy_base_url;
ALTER TABLE app_settings DROP COLUMN IF EXISTS ntfy_token;
ALTER TABLE app_settings DROP COLUMN IF EXISTS ntfy_priority;

DROP INDEX IF EXISTS ix_messages_send;
ALTER TABLE messages DROP COLUMN IF EXISTS send_status;
