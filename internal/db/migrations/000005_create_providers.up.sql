CREATE TABLE IF NOT EXISTS user_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL DEFAULT '',
    config_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'inactive',
    UNIQUE (user_id, provider)
);
