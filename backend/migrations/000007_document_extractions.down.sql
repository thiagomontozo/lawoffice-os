DROP TRIGGER IF EXISTS document_version_extraction_trigger ON document_versions;
DROP FUNCTION IF EXISTS enqueue_document_extraction();
DROP TABLE IF EXISTS document_extraction_pages;
DROP TABLE IF EXISTS document_extractions;
