-- Content themes. Sources opt into listening independently.
ALTER TABLE channels ALTER COLUMN disabled SET DEFAULT TRUE;
UPDATE channels SET disabled = TRUE;

CREATE TABLE themes (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    slug       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE theme_sources (
    theme_id     BIGINT NOT NULL REFERENCES themes(id) ON DELETE CASCADE,
    source_kind  TEXT NOT NULL CHECK (source_kind IN ('telegram', 'rss')),
    channel_id   BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    feed_id      BIGINT REFERENCES feeds(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT theme_sources_shape CHECK (
        (source_kind = 'telegram' AND channel_id IS NOT NULL AND feed_id IS NULL)
        OR (source_kind = 'rss' AND feed_id IS NOT NULL AND channel_id IS NULL)
    )
);

CREATE UNIQUE INDEX ux_theme_sources_channel ON theme_sources (theme_id, channel_id) WHERE channel_id IS NOT NULL;
CREATE UNIQUE INDEX ux_theme_sources_feed ON theme_sources (theme_id, feed_id) WHERE feed_id IS NOT NULL;
CREATE INDEX ix_theme_sources_channel ON theme_sources (channel_id) WHERE channel_id IS NOT NULL;
CREATE INDEX ix_theme_sources_feed ON theme_sources (feed_id) WHERE feed_id IS NOT NULL;
