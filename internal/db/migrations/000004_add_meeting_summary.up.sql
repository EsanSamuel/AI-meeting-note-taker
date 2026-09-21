-- 000004_add_meeting_summary.up.sql
ALTER TABLE meetings
ADD COLUMN IF NOT EXISTS summary TEXT;