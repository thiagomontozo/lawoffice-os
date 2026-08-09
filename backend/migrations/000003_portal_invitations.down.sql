DROP TABLE IF EXISTS portal_invitations;
DELETE FROM portal_users WHERE password_hash IS NULL;
ALTER TABLE portal_users ALTER COLUMN password_hash SET NOT NULL;
