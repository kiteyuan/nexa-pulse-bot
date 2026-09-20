-- Identity is (source_kind, channel_id|feed_id, external_id). Telegram message id is not a column.
UPDATE messages
SET external_id = telegram_message_id::text
WHERE source_kind = 'telegram'
  AND telegram_message_id IS NOT NULL
  AND (external_id IS NULL OR external_id = '');

UPDATE messages SET source_kind = 'rss' WHERE feed_id IS NOT NULL;
UPDATE messages SET source_kind = 'telegram' WHERE channel_id IS NOT NULL AND source_kind IS DISTINCT FROM 'rss';

ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_channel_id_telegram_message_id_key;
DROP INDEX IF EXISTS ux_messages_feed_item;

ALTER TABLE messages DROP COLUMN IF EXISTS telegram_message_id;

DELETE FROM messages
WHERE source_kind NOT IN ('telegram', 'rss')
   OR (source_kind = 'telegram' AND (channel_id IS NULL OR feed_id IS NOT NULL OR external_id = ''))
   OR (source_kind = 'rss' AND (feed_id IS NULL OR channel_id IS NOT NULL OR external_id = ''));

ALTER TABLE messages ADD CONSTRAINT messages_source_shape CHECK (
    (source_kind = 'telegram' AND channel_id IS NOT NULL AND feed_id IS NULL AND external_id <> '')
    OR (source_kind = 'rss' AND feed_id IS NOT NULL AND channel_id IS NULL AND external_id <> '')
);

CREATE UNIQUE INDEX ux_messages_telegram_item ON messages (channel_id, external_id) WHERE channel_id IS NOT NULL;
CREATE UNIQUE INDEX ux_messages_rss_item ON messages (feed_id, external_id) WHERE feed_id IS NOT NULL;

DELETE FROM ntfy_routes r
WHERE r.source_kind = 'telegram' AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.id = r.source_id);
DELETE FROM ntfy_routes r
WHERE r.source_kind = 'rss' AND NOT EXISTS (SELECT 1 FROM feeds f WHERE f.id = r.source_id);

ALTER TABLE ntfy_routes ADD COLUMN channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE;
ALTER TABLE ntfy_routes ADD COLUMN feed_id BIGINT REFERENCES feeds(id) ON DELETE CASCADE;

UPDATE ntfy_routes SET channel_id = source_id WHERE source_kind = 'telegram';
UPDATE ntfy_routes SET feed_id = source_id WHERE source_kind = 'rss';

ALTER TABLE ntfy_routes DROP CONSTRAINT IF EXISTS ntfy_routes_source_kind_source_id_key;
DROP INDEX IF EXISTS ntfy_routes_source_kind_source_id_key;
ALTER TABLE ntfy_routes DROP COLUMN source_id;

ALTER TABLE ntfy_routes ADD CONSTRAINT ntfy_routes_shape CHECK (
    (source_kind = 'telegram' AND channel_id IS NOT NULL AND feed_id IS NULL)
    OR (source_kind = 'rss' AND feed_id IS NOT NULL AND channel_id IS NULL)
);

CREATE UNIQUE INDEX ux_ntfy_routes_channel ON ntfy_routes (channel_id) WHERE channel_id IS NOT NULL;
CREATE UNIQUE INDEX ux_ntfy_routes_feed ON ntfy_routes (feed_id) WHERE feed_id IS NOT NULL;
