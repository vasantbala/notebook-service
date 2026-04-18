ALTER TABLE conversations ADD COLUMN rag_enabled   BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE conversations ADD COLUMN use_reasoning BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE conversations ADD COLUMN model         TEXT;     -- NULL = use service default
