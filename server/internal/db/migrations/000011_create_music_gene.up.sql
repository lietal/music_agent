CREATE TABLE IF NOT EXISTS music_gene (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    gene_type TEXT NOT NULL,
    gene_value TEXT NOT NULL,
    weight FLOAT DEFAULT 0.1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, gene_type, gene_value)
);
CREATE INDEX IF NOT EXISTS idx_music_gene_user ON music_gene(user_id, gene_type);
