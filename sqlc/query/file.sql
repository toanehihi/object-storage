-- name: CreateFile :exec
INSERT INTO files (id, owner_id, filename, object_key, size, content_type, status, checksum, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetFileByID :one
SELECT id, owner_id, filename, object_key, size, content_type, status, checksum, created_at, updated_at, scan_result, scanned_at
FROM files
WHERE id = $1;

-- name: GetFileByIDAndOwner :one
SELECT id, owner_id, filename, object_key, size, content_type, status, checksum, created_at, updated_at, scan_result, scanned_at
FROM files
WHERE id = $1 AND owner_id = $2 AND status != 'DELETED';

-- name: UpdateFileStatus :exec
UPDATE files SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteFile :exec
UPDATE files SET status = 'DELETED', updated_at = NOW()
WHERE id = $1 AND owner_id = $2;

-- name: ListFilesByOwner :many
SELECT id, owner_id, filename, object_key, size, content_type, status, checksum, created_at, updated_at, scan_result, scanned_at
FROM files
WHERE owner_id = $1 AND status != 'DELETED'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateFileScanResult :exec
UPDATE files
SET status = $2, scan_result = $3, scanned_at = $4, updated_at = NOW()
WHERE id = $1;
