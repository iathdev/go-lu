CREATE TABLE IF NOT EXISTS folders (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at  TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_folders_user_id ON folders(user_id);
CREATE INDEX IF NOT EXISTS idx_folders_deleted_at ON folders(deleted_at);

CREATE TABLE IF NOT EXISTS folder_vocabularies (
    folder_id     UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    vocabulary_id UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    added_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (folder_id, vocabulary_id)
);
