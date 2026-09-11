-- 000002_transcripts.up.sql
CREATE TABLE
    transcript_segments (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
        start_time DOUBLE PRECISION NOT NULL,
        end_time DOUBLE PRECISION NOT NULL,
        speaker TEXT NOT NULL,
        text TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_transcript_segments_meeting ON transcript_segments (meeting_id, start_time);