# Permission-aware Matter RAG

## Status

Accepted

## Context

Legal documents are sensitive, versioned and frequently restricted to a subset of a firm. A general chatbot or cross-tenant vector index could disclose data even when the ordinary Matter UI is secure. The product also needs useful document search when a generative provider is disabled.

## Decision

Derived text is chunked per immutable document version and stored in PostgreSQL with firm, Matter, document and page lineage. Retrieval repeats backend Matter authorization and only considers current, non-deleted versions. PostgreSQL full-text search is always available; optional embeddings add bounded in-process semantic reranking. Generation is provider-abstracted, disabled by default, context-bounded and returns explicit citations.

## Consequences

The security boundary stays with existing PostgreSQL authorization and no new vector service is required. Lexical retrieval degrades gracefully. Semantic ranking over a bounded candidate set will not scale as far as a dedicated vector index, and enabling an external provider requires a deployment-specific confidentiality and data-processing review.
