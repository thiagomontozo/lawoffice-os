CREATE TABLE realtime_events (
    id bigserial PRIMARY KEY,
    firm_id uuid NOT NULL REFERENCES firms(id) ON DELETE CASCADE,
    event_type varchar(100) NOT NULL,
    resource_type varchar(80) NOT NULL,
    resource_id text NOT NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL DEFAULT (now() + interval '7 days')
);

CREATE INDEX realtime_events_firm_replay_idx ON realtime_events(firm_id, id);
CREATE INDEX realtime_events_expiry_idx ON realtime_events(expires_at);
