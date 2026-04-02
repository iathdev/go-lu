CREATE TABLE vocabularies (
    id                   UUID PRIMARY KEY,
    language_id          UUID NOT NULL,
    level_id             UUID,
    writing_level_id     UUID,
    word                 VARCHAR(255) NOT NULL,
    phonetic             VARCHAR(255),
    audio_url            VARCHAR(500),
    image_url            VARCHAR(500),
    frequency_rank       INTEGER,
    metadata             JSONB DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_vocab_word_lang ON vocabularies(language_id, word) WHERE deleted_at IS NULL;
CREATE INDEX idx_vocab_language ON vocabularies(language_id);
CREATE INDEX idx_vocab_level ON vocabularies(level_id);
CREATE INDEX idx_vocab_writing_level ON vocabularies(writing_level_id);
CREATE INDEX idx_vocab_frequency ON vocabularies(frequency_rank) WHERE frequency_rank IS NOT NULL;
CREATE INDEX idx_vocab_metadata ON vocabularies USING GIN (metadata);
