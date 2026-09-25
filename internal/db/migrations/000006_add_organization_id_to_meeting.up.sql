-- 000006_add_organization_id_to_meeting.up.sql
ALTER TABLE meetings
ADD COLUMN organization_id UUID
REFERENCES organizations(id)
ON DELETE CASCADE;

CREATE INDEX idx_meetings_organization_id
ON meetings(organization_id);