CREATE TABLE IF NOT EXISTS vocabularies (
    id               UUID PRIMARY KEY,
    hanzi            VARCHAR(255) NOT NULL,
    pinyin           VARCHAR(255) NOT NULL,
    meaning_vi       TEXT,
    meaning_en       TEXT,
    hsk_level        INTEGER NOT NULL,
    audio_url        VARCHAR(500),
    examples         JSONB DEFAULT '[]'::jsonb,
    radicals         TEXT[] DEFAULT '{}',
    stroke_count     INTEGER,
    stroke_data_url  VARCHAR(500),
    recognition_only BOOLEAN DEFAULT false,
    frequency_rank   INTEGER,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at       TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_vocabularies_hsk_level ON vocabularies(hsk_level);
CREATE INDEX IF NOT EXISTS idx_vocabularies_deleted_at ON vocabularies(deleted_at);
CREATE INDEX IF NOT EXISTS idx_vocabularies_examples ON vocabularies USING GIN (examples);
CREATE INDEX IF NOT EXISTS idx_vocabularies_frequency_rank ON vocabularies(frequency_rank) WHERE frequency_rank IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_vocabularies_hanzi_unique ON vocabularies(hanzi) WHERE deleted_at IS NULL;
