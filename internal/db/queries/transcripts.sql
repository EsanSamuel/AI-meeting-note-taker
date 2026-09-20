-- db/queries/transcripts.sql
-- name: CreateTranscriptSegment :one
INSERT INTO
    transcript_segments (
        meeting_id,
        start_time,
        end_time,
        speaker,
        speaker_id,
        text
    )
VALUES
    ($1, $2, $3, $4, $5, $6)
RETURNING
    *;

-- name: GetTranscriptSegment :one
SELECT
    *
FROM
    transcript_segments
WHERE
    id = $1
LIMIT
    1;

-- name: ListTranscriptSegments :many
SELECT
    *
FROM
    transcript_segments
WHERE
    meeting_id = $1
ORDER BY
    start_time ASC;

-- name: UpdateTranscriptSegment :one
UPDATE transcript_segments
SET
    text = $2,
    start_time = $3,
    end_time = $4,
    speaker = $5,
    updated_at = NOW()
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteTranscriptSegment :exec
DELETE FROM transcript_segments
WHERE
    id = $1;

-- name: DeleteTranscriptSegmentsByMeeting :exec
DELETE FROM transcript_segments
WHERE
    meeting_id = $1;

-- name: UpdateSpeakers :many
UPDATE transcript_segments
SET
    speaker = $1
WHERE
    meeting_id = $2
    AND speaker_id = $3
RETURNING
    *;