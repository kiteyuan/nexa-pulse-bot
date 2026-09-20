ALTER TABLE channels ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE app_settings RENAME COLUMN telegram_poll_interval_seconds TO collect_interval_seconds;
ALTER TABLE app_settings ADD COLUMN ntfy_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE ntfy_routes (
    id           BIGSERIAL PRIMARY KEY,
    source_kind  TEXT NOT NULL CHECK (source_kind IN ('telegram', 'rss')),
    source_id    BIGINT NOT NULL,
    topic        TEXT NOT NULL REFERENCES topics(name) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_kind, source_id)
);

INSERT INTO ntfy_routes(source_kind, source_id, topic)
SELECT 'telegram', c.id, c.ntfy_topic
FROM channels c
JOIN topics t ON t.name = c.ntfy_topic
WHERE c.ntfy_topic <> ''
ON CONFLICT (source_kind, source_id) DO NOTHING;

INSERT INTO ntfy_routes(source_kind, source_id, topic)
SELECT 'rss', f.id, f.ntfy_topic
FROM feeds f
JOIN topics t ON t.name = f.ntfy_topic
WHERE f.ntfy_topic <> ''
ON CONFLICT (source_kind, source_id) DO NOTHING;

UPDATE app_settings SET ntfy_enabled = TRUE WHERE EXISTS (SELECT 1 FROM ntfy_routes);

UPDATE messages SET send_status = 'idle' WHERE send_status = 'ready';

ALTER TABLE channels DROP COLUMN ntfy_topic;
ALTER TABLE feeds DROP COLUMN ntfy_topic;
