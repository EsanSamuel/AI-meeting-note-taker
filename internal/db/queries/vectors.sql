-- db/queries/vectors.sql
-- name: CreateTranscriptVectorEmbedding :one
INSERT INTO
    transcript_chunk (meeting_id, chunk, embedding)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: SearchTranscriptChunk :many
SELECT
    id,
    meeting_id,
    chunk,
    embedding <=> $1 AS distance
FROM
    transcript_chunk
ORDER BY
    embedding <=> $1
LIMIT
    $2;