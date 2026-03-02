ALTER TABLE users
    DROP COLUMN IF EXISTS llm_preferred_provider,
    DROP COLUMN IF EXISTS llm_preferred_model;
