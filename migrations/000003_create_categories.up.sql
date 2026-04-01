CREATE TABLE categories (
    id               UUID PRIMARY KEY,
    language_id      UUID NOT NULL,
    prep_category_id INT,
    code             VARCHAR(20) NOT NULL,
    name             VARCHAR(100) NOT NULL,
    is_public        BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(language_id, code)
);

CREATE INDEX idx_categories_language ON categories(language_id);
CREATE UNIQUE INDEX idx_categories_prep_category ON categories(prep_category_id) WHERE prep_category_id IS NOT NULL;
