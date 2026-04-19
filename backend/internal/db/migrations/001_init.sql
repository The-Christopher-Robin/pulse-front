CREATE TABLE IF NOT EXISTS products (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    image_url   TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS experiments (
    id          SERIAL PRIMARY KEY,
    key         TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('draft','running','paused','ended')),
    salt        TEXT NOT NULL,
    traffic_pct INTEGER NOT NULL CHECK (traffic_pct BETWEEN 0 AND 100),
    variants    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exposures (
    id             BIGSERIAL PRIMARY KEY,
    experiment_key TEXT NOT NULL,
    variant_key    TEXT NOT NULL,
    user_id        TEXT NOT NULL,
    occurred_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS exposures_experiment_idx
    ON exposures (experiment_key, occurred_at DESC);

CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    target_id   TEXT,
    properties  JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS events_type_time_idx
    ON events (event_type, occurred_at DESC);
