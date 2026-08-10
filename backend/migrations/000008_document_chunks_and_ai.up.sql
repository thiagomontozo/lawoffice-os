ALTER TABLE document_extractions
    ADD CONSTRAINT document_extractions_id_firm_unique UNIQUE (id, firm_id);
ALTER TABLE document_extraction_pages
    ADD CONSTRAINT document_extraction_pages_extraction_firm_fk
    FOREIGN KEY (extraction_id, firm_id) REFERENCES document_extractions(id, firm_id) ON DELETE CASCADE;

CREATE TABLE document_chunks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id uuid NOT NULL,
    matter_id uuid,
    document_id uuid NOT NULL,
    document_version_id uuid NOT NULL,
    extraction_id uuid NOT NULL,
    page_number integer NOT NULL CHECK (page_number > 0),
    chunk_index integer NOT NULL CHECK (chunk_index >= 0),
    content text NOT NULL CHECK (length(content) > 0),
    content_hash text NOT NULL CHECK (length(content_hash) = 64),
    character_count integer NOT NULL CHECK (character_count > 0),
    embedding jsonb,
    embedding_model text,
    embedding_status text NOT NULL DEFAULT 'pending'
        CHECK (embedding_status IN ('pending', 'processing', 'succeeded', 'failed')),
    embedding_attempts integer NOT NULL DEFAULT 0 CHECK (embedding_attempts >= 0),
    embedding_available_at timestamptz NOT NULL DEFAULT now(),
    embedding_locked_at timestamptz,
    embedding_locked_by text,
    embedding_error_code text,
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('portuguese', content)) STORED,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (firm_id, document_version_id, page_number, chunk_index),
    FOREIGN KEY (matter_id, firm_id) REFERENCES matters(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (document_id, firm_id) REFERENCES documents(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (document_version_id, firm_id) REFERENCES document_versions(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (extraction_id, firm_id) REFERENCES document_extractions(id, firm_id) ON DELETE CASCADE
);

CREATE INDEX document_chunks_firm_matter_idx
    ON document_chunks (firm_id, matter_id, document_id);
CREATE INDEX document_chunks_version_idx
    ON document_chunks (firm_id, document_version_id);
CREATE INDEX document_chunks_search_idx
    ON document_chunks USING gin (search_vector);
CREATE INDEX document_chunks_embedding_queue_idx
    ON document_chunks (embedding_status, embedding_available_at)
    WHERE embedding_status IN ('pending', 'processing');

CREATE TABLE ai_responses (
    id uuid PRIMARY KEY,
    firm_id uuid NOT NULL,
    user_id uuid NOT NULL,
    matter_id uuid NOT NULL,
    model text NOT NULL,
    retrieval text NOT NULL CHECK (retrieval IN ('keyword', 'hybrid')),
    citation_count integer NOT NULL CHECK (citation_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, firm_id, user_id, matter_id),
    FOREIGN KEY (user_id, firm_id) REFERENCES users(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (matter_id, firm_id) REFERENCES matters(id, firm_id) ON DELETE CASCADE
);

CREATE INDEX ai_responses_firm_matter_idx ON ai_responses (firm_id, matter_id, created_at DESC);

CREATE TABLE ai_feedback (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id uuid NOT NULL,
    user_id uuid NOT NULL,
    matter_id uuid NOT NULL,
    response_id uuid NOT NULL,
    rating text NOT NULL CHECK (rating IN ('helpful', 'not_helpful')),
    reason text CHECK (reason IS NULL OR length(reason) <= 500),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (firm_id, user_id, response_id),
    FOREIGN KEY (user_id, firm_id) REFERENCES users(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (matter_id, firm_id) REFERENCES matters(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (response_id, firm_id, user_id, matter_id)
        REFERENCES ai_responses(id, firm_id, user_id, matter_id) ON DELETE CASCADE
);

CREATE INDEX ai_feedback_firm_matter_idx ON ai_feedback (firm_id, matter_id, created_at DESC);
