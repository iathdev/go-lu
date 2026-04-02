CREATE TABLE grammar_points (
    id                   UUID PRIMARY KEY,
    category_id          UUID NOT NULL,
    level_id             UUID,
    code                 VARCHAR(50) NOT NULL,
    pattern              VARCHAR(255) NOT NULL,
    examples             JSONB DEFAULT '{}'::jsonb,
    rule                 JSONB DEFAULT '{}'::jsonb,
    common_mistakes      JSONB DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(category_id, code)
);

CREATE INDEX idx_gp_category ON grammar_points(category_id);
CREATE INDEX idx_gp_level ON grammar_points(level_id);
