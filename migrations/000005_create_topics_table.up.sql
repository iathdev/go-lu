CREATE TABLE IF NOT EXISTS topics (
    id         UUID PRIMARY KEY,
    name_cn    VARCHAR(100) NOT NULL,
    name_vi    VARCHAR(100) NOT NULL,
    name_en    VARCHAR(100) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vocabulary_topics (
    vocabulary_id UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    topic_id      UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (vocabulary_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_vocabulary_topics_topic_id ON vocabulary_topics(topic_id);
