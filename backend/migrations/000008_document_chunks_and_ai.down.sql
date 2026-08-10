DROP TABLE IF EXISTS ai_feedback;
DROP TABLE IF EXISTS ai_responses;
DROP TABLE IF EXISTS document_chunks;
ALTER TABLE document_extraction_pages
    DROP CONSTRAINT IF EXISTS document_extraction_pages_extraction_firm_fk;
ALTER TABLE document_extractions
    DROP CONSTRAINT IF EXISTS document_extractions_id_firm_unique;
