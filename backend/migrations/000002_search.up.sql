CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_matters_title_trgm ON matters USING gin (title gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_clients_name_trgm ON clients USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_contacts_name_trgm ON contacts USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_title_trgm ON documents USING gin (title gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_matter_parties_name_trgm ON matter_parties USING gin (name gin_trgm_ops);
CREATE INDEX idx_tags_name_trgm ON tags USING gin (name gin_trgm_ops);
