CREATE TABLE accounts (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    api_id      INTEGER NOT NULL,
    api_hash    TEXT NOT NULL,
    phone       TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'offline',
    last_sync   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE channels (
    id               BIGSERIAL PRIMARY KEY,
    account_id       BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    telegram_id      BIGINT NOT NULL,
    access_hash      BIGINT NOT NULL DEFAULT 0,
    username         TEXT NOT NULL DEFAULT '',
    title            TEXT NOT NULL DEFAULT '',
    ntfy_topic       TEXT NOT NULL DEFAULT '',
    last_message_id  BIGINT NOT NULL DEFAULT 0,
    UNIQUE (account_id, telegram_id)
);

CREATE INDEX ix_channels_topic ON channels (ntfy_topic);

CREATE TABLE messages (
    id                   BIGSERIAL PRIMARY KEY,
    channel_id           BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    telegram_message_id  BIGINT NOT NULL,
    content              TEXT NOT NULL DEFAULT '',
    content_hash         TEXT NOT NULL,
    media_paths          JSONB NOT NULL DEFAULT '[]'::jsonb,
    llm_status           TEXT NOT NULL DEFAULT 'pending',
    send_status          TEXT NOT NULL DEFAULT 'idle',
    llm_result           JSONB,
    importance           DOUBLE PRECISION,
    error_message        TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_id, telegram_message_id)
);

CREATE INDEX ix_messages_llm ON messages (llm_status);
CREATE INDEX ix_messages_send ON messages (send_status);
CREATE INDEX ix_messages_hash ON messages (content_hash);

CREATE TABLE topics (
    name        TEXT PRIMARY KEY,
    disabled    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_settings (
    id                              INTEGER PRIMARY KEY CHECK (id = 1),
    llm_enabled                     BOOLEAN NOT NULL DEFAULT FALSE,
    llm_base_url                    TEXT NOT NULL DEFAULT 'https://api.deepseek.com/v1',
    llm_api_key                     TEXT NOT NULL DEFAULT '',
    llm_model                       TEXT NOT NULL DEFAULT 'deepseek-v4-flash',
    llm_temperature                 DOUBLE PRECISION NOT NULL DEFAULT 0.3,
    translate_to                    TEXT NOT NULL DEFAULT 'off',
    min_length                      INTEGER NOT NULL DEFAULT 20,
    block_keywords                  JSONB NOT NULL DEFAULT '["广告","加群","优惠券"]'::jsonb,
    ntfy_base_url                   TEXT NOT NULL DEFAULT 'http://127.0.0.1:2586',
    ntfy_token                      TEXT NOT NULL DEFAULT '',
    ntfy_priority                   INTEGER NOT NULL DEFAULT 3,
    poll_interval_seconds           DOUBLE PRECISION NOT NULL DEFAULT 2,
    telegram_poll_interval_seconds  DOUBLE PRECISION NOT NULL DEFAULT 1800
);

INSERT INTO app_settings (id) VALUES (1);

CREATE TABLE runtime_logs (
    id          BIGSERIAL PRIMARY KEY,
    level       TEXT NOT NULL DEFAULT 'INFO',
    source      TEXT NOT NULL DEFAULT 'system',
    message     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_logs_created ON runtime_logs (created_at DESC);
