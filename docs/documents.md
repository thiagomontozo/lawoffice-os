# Documents

`Document` is the durable metadata identity. `DocumentVersion` is an immutable file revision with original display filename, generated storage key, MIME, byte size, SHA-256 checksum, creator, timestamp and notes. `currentVersionId` selects the active revision without destroying history.

## Upload

1. Authenticate and require document permission.
2. Check the target firm and Matter write access.
3. Bound request size and validate MIME/extension.
4. Generate an opaque UUID storage key; firm authorization remains in PostgreSQL metadata.
5. Write the object atomically.
6. Insert version metadata and advance the pointer in one database transaction.
7. Remove the new object when metadata insertion fails.

## Download

Clients request a document ID and optional version number, never a storage path. The repository verifies firm, soft-delete state and Matter access before returning an internal key. The API returns `nosniff` and a sanitized attachment filename.

Folders provide simple Matter organization without pretending to be a general filesystem. Soft deletion hides metadata but does not immediately remove objects; a future retention worker must reconcile audit, legal hold and storage lifecycle.
