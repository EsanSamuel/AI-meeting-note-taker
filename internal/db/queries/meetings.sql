-- db/queries/meetings.sql
-- name: CreateMeeting :one
INSERT INTO
    meetings (
        title,
        started_at,
        ended_at,
        duration_seconds,
        audio_path,
        video_path
    )
VALUES
    ($1, $2, $3, $4, $5, $6)
RETURNING
    *;

-- name: GetMeeting :one
SELECT
    *
FROM
    meetings
WHERE
    id = $1
LIMIT
    1;

-- name: ListMeetings :many
SELECT
    *
FROM
    meetings
ORDER BY
    created_at DESC;

-- name: ListMeetingsWithDecisionsAndActionItems :many
SELECT
    *
FROM
    meetings
    JOIN meeting_decisions ON meetings.id = meeting_decisions.meeting_id
    JOIN meeting_action_items ON meetings.id = meeting_action_items.meeting_id
ORDER BY
    meetings.created_at DESC;

-- name: UpdateMeeting :one
UPDATE meetings
SET
    title = $2,
    started_at = $3,
    ended_at = $4,
    duration_seconds = $5,
    audio_path = $6,
    video_path = $7,
    updated_at = NOW()
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteMeeting :exec
DELETE FROM meetings
WHERE
    id = $1;

-- name: CreateMeetingDecision :one
INSERT INTO
    meeting_decisions (meeting_id, decision, timestamp_seconds)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetMeetingDecision :one
SELECT
    *
FROM
    meeting_decisions
WHERE
    id = $1
LIMIT
    1;

-- name: ListMeetingDecisions :many
SELECT
    *
FROM
    meeting_decisions
WHERE
    meeting_id = $1
ORDER BY
    timestamp_seconds ASC;

-- name: DeleteMeetingDecision :exec
DELETE FROM meeting_decisions
WHERE
    id = $1;

-- name: DeleteMeetingDecisions :exec
DELETE FROM meeting_decisions
WHERE
    meeting_id = $1;

-- name: CreateMeetingActionItem :one
INSERT INTO
    meeting_action_items (meeting_id, task, assignee, timestamp_seconds)
VALUES
    ($1, $2, $3, $4)
RETURNING
    *;

-- name: GetMeetingActionItem :one
SELECT
    *
FROM
    meeting_action_items
WHERE
    id = $1
LIMIT
    1;

-- name: ListMeetingActionItems :many
SELECT
    *
FROM
    meeting_action_items
WHERE
    meeting_id = $1
ORDER BY
    timestamp_seconds ASC;

-- name: ListIncompleteActionItems :many
SELECT
    *
FROM
    meeting_action_items
WHERE
    meeting_id = $1
    AND completed = FALSE
ORDER BY
    timestamp_seconds ASC;

-- name: UpdateMeetingActionItem :one
UPDATE meeting_action_items
SET
    task = $2,
    assignee = $3,
    timestamp_seconds = $4,
    completed = $5
WHERE
    id = $1
RETURNING
    *;

-- name: MarkActionItemCompleted :one
UPDATE meeting_action_items
SET
    completed = TRUE
WHERE
    id = $1
RETURNING
    *;

-- name: MarkActionItemIncomplete :one
UPDATE meeting_action_items
SET
    completed = FALSE
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteMeetingActionItem :exec
DELETE FROM meeting_action_items
WHERE
    id = $1;

-- name: DeleteMeetingActionItems :exec
DELETE FROM meeting_action_items
WHERE
    meeting_id = $1;