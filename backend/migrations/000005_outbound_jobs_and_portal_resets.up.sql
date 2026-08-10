CREATE TABLE outbound_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id uuid NOT NULL REFERENCES firms(id) ON DELETE CASCADE,
    job_type varchar(80) NOT NULL CHECK (job_type IN ('email.send')),
    encrypted_payload bytea,
    status varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','completed','failed')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 20),
    available_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    locked_by varchar(120),
    last_error varchar(500),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);

CREATE INDEX outbound_jobs_pending_idx ON outbound_jobs(status, available_at, created_at)
    WHERE status IN ('pending','processing');

CREATE TABLE portal_password_resets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id uuid NOT NULL,
    portal_user_id uuid NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (portal_user_id, firm_id) REFERENCES portal_users(id, firm_id) ON DELETE CASCADE
);

CREATE INDEX portal_password_resets_lookup_idx ON portal_password_resets(token_hash, expires_at)
    WHERE used_at IS NULL;
