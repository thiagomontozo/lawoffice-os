ALTER TABLE outbound_jobs ADD COLUMN deduplication_key varchar(200);

CREATE UNIQUE INDEX outbound_jobs_deduplication_idx
    ON outbound_jobs(deduplication_key)
    WHERE deduplication_key IS NOT NULL;

ALTER TABLE notifications ADD COLUMN email_queued_at timestamptz;

CREATE TABLE user_notification_preferences (
    firm_id uuid NOT NULL,
    user_id uuid NOT NULL,
    email_deadlines boolean NOT NULL DEFAULT false,
    email_tasks boolean NOT NULL DEFAULT false,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (firm_id, user_id),
    FOREIGN KEY (user_id, firm_id) REFERENCES users(id, firm_id) ON DELETE CASCADE
);

CREATE INDEX notifications_email_pending_idx
    ON notifications(created_at)
    WHERE email_queued_at IS NULL;
