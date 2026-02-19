-- name: CreateBinaryBlob :one
INSERT INTO binary_blobs (filename, content_type, size_bytes, storage_path)
VALUES (?, ?, ?, ?)
RETURNING id, filename, content_type, size_bytes, storage_path, created_at;

-- name: GetBinaryBlobByID :one
SELECT id, filename, content_type, size_bytes, storage_path, created_at
FROM binary_blobs WHERE id = ?;

-- name: DeleteBinaryBlob :exec
DELETE FROM binary_blobs WHERE id = ?;
