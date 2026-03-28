CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notebook_id UUID NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT 'New Conversation',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content         TEXT NOT NULL,
    token_count     INT  NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sources (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notebook_id UUID NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    filename    TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    mime_type   TEXT NOT NULL DEFAULT 'application/octet-stream',
    status      TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','processing','ready','failed')),
    chunk_count INT  NOT NULL DEFAULT 0,
    -- rag_doc_id is the doc_id issued by rag-anything on ingest; used to filter
    -- Qdrant results when calling the /retrieve endpoint.
    rag_doc_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE citations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id  UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    source_id   UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    chunk_index INT  NOT NULL,
    score       FLOAT NOT NULL DEFAULT 0
);

CREATE INDEX idx_conversations_notebook_id ON conversations(notebook_id);
CREATE INDEX idx_messages_conversation_id  ON messages(conversation_id);
CREATE INDEX idx_sources_notebook_id       ON sources(notebook_id);