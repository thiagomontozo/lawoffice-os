CREATE TABLE document_extractions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id uuid NOT NULL,
    document_id uuid NOT NULL,
    document_version_id uuid NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','succeeded','failed','unsupported')),
    provider varchar(80),
    language varchar(20),
    page_count integer NOT NULL DEFAULT 0 CHECK (page_count >= 0),
    average_confidence numeric(6,5),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 20),
    available_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    locked_by varchar(120),
    error_code varchar(80),
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    completed_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(document_version_id),
    FOREIGN KEY (document_id, firm_id) REFERENCES documents(id, firm_id) ON DELETE CASCADE,
    FOREIGN KEY (document_version_id, firm_id) REFERENCES document_versions(id, firm_id) ON DELETE CASCADE
);

CREATE INDEX document_extractions_pending_idx ON document_extractions(status, available_at, created_at)
    WHERE status IN ('pending','processing');
CREATE INDEX document_extractions_document_idx ON document_extractions(firm_id, document_id, created_at DESC);

CREATE TABLE document_extraction_pages (
    extraction_id uuid NOT NULL REFERENCES document_extractions(id) ON DELETE CASCADE,
    firm_id uuid NOT NULL REFERENCES firms(id) ON DELETE CASCADE,
    page_number integer NOT NULL CHECK (page_number > 0),
    content text NOT NULL,
    confidence numeric(6,5),
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('portuguese', content)) STORED,
    PRIMARY KEY (extraction_id, page_number)
);

CREATE INDEX document_extraction_pages_search_idx ON document_extraction_pages USING GIN(search_vector);

CREATE FUNCTION enqueue_document_extraction() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO document_extractions(firm_id, document_id, document_version_id)
    VALUES(NEW.firm_id, NEW.document_id, NEW.id)
    ON CONFLICT(document_version_id) DO NOTHING;
    RETURN NEW;
END;
$$;

CREATE TRIGGER document_version_extraction_trigger
    AFTER INSERT ON document_versions
    FOR EACH ROW EXECUTE FUNCTION enqueue_document_extraction();

INSERT INTO document_extractions(firm_id, document_id, document_version_id)
SELECT firm_id, document_id, id FROM document_versions
ON CONFLICT(document_version_id) DO NOTHING;
