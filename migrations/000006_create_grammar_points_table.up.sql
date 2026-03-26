CREATE TABLE IF NOT EXISTS grammar_points (
    id             UUID PRIMARY KEY,
    code           VARCHAR(50) NOT NULL UNIQUE,
    pattern        VARCHAR(255) NOT NULL,
    example_cn     TEXT,
    example_vi     TEXT,
    rule           TEXT,
    common_mistake TEXT,
    hsk_level      INTEGER NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grammar_points_hsk_level ON grammar_points(hsk_level);

CREATE TABLE IF NOT EXISTS vocabulary_grammar_points (
    vocabulary_id    UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    PRIMARY KEY (vocabulary_id, grammar_point_id)
);

CREATE INDEX IF NOT EXISTS idx_vocabulary_grammar_points_gp_id ON vocabulary_grammar_points(grammar_point_id);
