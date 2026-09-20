CREATE TABLE feeds (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL UNIQUE,
    ntfy_topic  TEXT NOT NULL DEFAULT '',
    disabled    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE messages ALTER COLUMN channel_id DROP NOT NULL;
ALTER TABLE messages ALTER COLUMN telegram_message_id DROP NOT NULL;
ALTER TABLE messages ADD COLUMN source_kind TEXT NOT NULL DEFAULT 'telegram';
ALTER TABLE messages ADD COLUMN feed_id BIGINT REFERENCES feeds(id) ON DELETE CASCADE;
ALTER TABLE messages ADD COLUMN external_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN link TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN origin_title TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX ux_messages_feed_item ON messages (feed_id, external_id) WHERE feed_id IS NOT NULL;
