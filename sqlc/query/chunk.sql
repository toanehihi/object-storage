-- name: CreateChunk :exec
INSERT INTO file_chunks (id, file_id, chunk_index, uploaded, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: MarkChunkUploaded :exec
UPDATE file_chunks SET uploaded = TRUE, etag = $3
WHERE file_id = $1 AND chunk_index = $2;

-- name: GetUploadedChunkIndices :many
SELECT chunk_index
FROM file_chunks
WHERE file_id = $1 AND uploaded = TRUE
ORDER BY chunk_index;

-- name: CountChunksByFileID :one
SELECT COUNT(*)::int AS total
FROM file_chunks
WHERE file_id = $1;

-- name: CountUploadedChunksByFileID :one
SELECT COUNT(*)::int AS total
FROM file_chunks
WHERE file_id = $1 AND uploaded = TRUE;
