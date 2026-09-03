-- name: CreateDocument :one
INSERT INTO documents (owner_id, original_name, media_type, size_bytes, storage_key)
VALUES (sqlc.arg(owner_id), sqlc.arg(original_name), sqlc.arg(media_type), sqlc.arg(size_bytes), sqlc.arg(storage_key))
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListDocumentsByOwner :many
SELECT * FROM documents
WHERE owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL
ORDER BY id DESC LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountDocumentsByOwner :one
SELECT COUNT(*) FROM documents WHERE owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL;

-- name: ListAllDocuments :many
SELECT * FROM documents
WHERE deleted_at IS NULL
ORDER BY id DESC LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountAllDocuments :one
SELECT COUNT(*) FROM documents WHERE deleted_at IS NULL;

-- name: SoftDeleteDocument :execrows
UPDATE documents SET deleted_at = COALESCE(deleted_at, NOW()), updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ClaimDocumentProcessing :execrows
UPDATE documents SET status = 'processing', processing_error = NULL, updated_at = NOW()
WHERE id = sqlc.arg(id) AND status IN ('pending', 'failed') AND deleted_at IS NULL;

-- name: MarkDocumentReady :execrows
UPDATE documents
SET status = 'ready', checksum_sha256 = sqlc.arg(checksum_sha256), size_bytes = sqlc.arg(size_bytes), processing_error = NULL, updated_at = NOW()
WHERE id = sqlc.arg(id) AND status = 'processing' AND deleted_at IS NULL;

-- name: MarkDocumentFailed :execrows
UPDATE documents SET status = 'failed', processing_error = sqlc.arg(processing_error), updated_at = NOW()
WHERE id = sqlc.arg(id) AND status = 'processing' AND deleted_at IS NULL;
