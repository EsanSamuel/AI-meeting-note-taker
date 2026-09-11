-- 000001_create_meeting.up.sql
CREATE TABLE
    meetings (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        title TEXT NOT NULL,
        started_at TIMESTAMPTZ,
        ended_at TIMESTAMPTZ,
        duration_seconds DOUBLE PRECISION,
        audio_path TEXT,
        video_path TEXT,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE
    meeting_decisions (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
        decision TEXT NOT NULL,
        timestamp_seconds DOUBLE PRECISION NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_decisions_meeting ON meeting_decisions (meeting_id);

CREATE TABLE
    meeting_action_items (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
        task TEXT NOT NULL,
        assignee TEXT,
        timestamp_seconds DOUBLE PRECISION NOT NULL,
        completed BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_action_items_meeting ON meeting_action_items (meeting_id);