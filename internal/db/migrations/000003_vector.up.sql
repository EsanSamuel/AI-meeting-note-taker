-- 000001_create_meeting.up.sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE
    transcript_chunk (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
        chunk TEXT NOT NULL,
        embedding vector (384) NOT NULL
    );