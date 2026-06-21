CREATE TABLE short_links (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code ~ '^[0-9A-Za-z]{4,32}$'),
    original_url TEXT NOT NULL UNIQUE CHECK (length(original_url) > 0),
    title TEXT NOT NULL CHECK (length(title) > 0),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled', 'deleted')),
    created_by_telegram_id BIGINT NOT NULL,
    updated_by_telegram_id BIGINT,
    disabled_at TIMESTAMPTZ,
    disabled_by_telegram_id BIGINT,
    deleted_at TIMESTAMPTZ,
    deleted_by_telegram_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX short_links_visible_created_idx ON short_links (created_at DESC, id DESC)
    WHERE status IN ('active', 'disabled');
CREATE INDEX short_links_status_created_idx ON short_links (status, created_at DESC, id DESC);
CREATE INDEX short_links_created_idx ON short_links (created_at DESC, id DESC);

CREATE TABLE link_events (
    id BIGSERIAL PRIMARY KEY,
    short_link_id BIGINT REFERENCES short_links (id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    actor_telegram_id BIGINT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX link_events_short_link_created_idx ON link_events (short_link_id, created_at DESC);
CREATE INDEX link_events_type_created_idx ON link_events (event_type, created_at DESC);
