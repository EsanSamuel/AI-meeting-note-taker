-- 000004_add_meeting_summary.down.sql
ALTER TABLE meetings
DROP COLUMN IF EXISTS summary;