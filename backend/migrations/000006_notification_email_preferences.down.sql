DROP TABLE IF EXISTS user_notification_preferences;
DROP INDEX IF EXISTS notifications_email_pending_idx;
ALTER TABLE notifications DROP COLUMN IF EXISTS email_queued_at;
DROP INDEX IF EXISTS outbound_jobs_deduplication_idx;
ALTER TABLE outbound_jobs DROP COLUMN IF EXISTS deduplication_key;
