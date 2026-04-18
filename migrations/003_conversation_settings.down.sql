ALTER TABLE conversations DROP COLUMN IF EXISTS rag_enabled;
ALTER TABLE conversations DROP COLUMN IF EXISTS use_reasoning;
ALTER TABLE conversations DROP COLUMN IF EXISTS model;
