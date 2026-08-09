ALTER TABLE portal_users ALTER COLUMN password_hash DROP NOT NULL;

CREATE TABLE portal_invitations(
    token_hash bytea PRIMARY KEY,
    firm_id uuid NOT NULL REFERENCES firms(id) ON DELETE CASCADE,
    portal_user_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY(portal_user_id,firm_id) REFERENCES portal_users(id,firm_id) ON DELETE CASCADE,
    FOREIGN KEY(created_by,firm_id) REFERENCES users(id,firm_id) ON DELETE RESTRICT
);

CREATE INDEX idx_portal_invitations_expiry ON portal_invitations(firm_id,expires_at) WHERE accepted_at IS NULL;
