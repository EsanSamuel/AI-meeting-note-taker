-- db/queries/vectors.sql
-- name: CreateTranscriptVectorEmbedding :one
INSERT INTO
    transcript_chunk (meeting_id, chunk, embedding)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetTranscriptVector :many
SELECT
    *
FROM
    transcript_chunk
WHERE
    meeting_id = $1;

-- name: SearchTranscriptChunk :many
SELECT
    id,
    meeting_id,
    chunk,
    embedding <=> $1 AS distance
FROM
    transcript_chunk
WHERE
    meeting_id = $2
ORDER BY
    embedding <=> $1
LIMIT
    $3;