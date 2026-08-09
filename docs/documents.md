# Documents

`Document` is the durable metadata identity. `DocumentVersion` is an immutable file revision with original display filename, generated storage key, MIME, byte size, SHA-256 checksum, creator, timestamp and notes. `currentVersionId` selects the active revision without destroying history.

## Upload

1. Authenticate and require document permission.
2. Check the target firm and Matter write access.
3. Bound request size and validate detected MIME against the supplied extension before persistence.
4. Stream the body through a size counter and SHA-256 hasher instead of buffering the complete legal document in API memory.
5. Generate an opaque, firm-prefixed UUID storage key; firm authorization remains in PostgreSQL metadata.
6. Write the object atomically.
7. Insert version metadata and advance the pointer in one database transaction.
8. Remove the new object when metadata insertion fails.

## Download

Clients request a document ID and optional version number, never a storage path. The repository verifies firm, soft-delete state and Matter access before returning an internal key. The API returns `nosniff` and a sanitized attachment filename. Successful internal downloads create an audit event containing the document ID and version, never its contents.

## Deletion, recovery and retention

Deleting a document is a reversible metadata operation. The active library hides the document, records `deletedAt`/`deletedBy`, emits audit and realtime events, and retains every immutable version in object storage. An authorized administrator can restore the metadata without reconstructing the version chain.

Physical object deletion is intentionally separate. A future retention worker must consider legal hold, firm policy and auditable approval before purging any version. This prevents a routine UI action from silently destroying legal evidence.

Folders provide simple Matter organization without pretending to be a general filesystem. The local adapter remains intended for single-instance or shared-volume deployments; the storage interface is the boundary for a future S3/MinIO adapter.
