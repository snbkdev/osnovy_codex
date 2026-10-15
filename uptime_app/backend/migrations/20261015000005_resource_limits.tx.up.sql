CREATE TABLE rate_limits (
 key text PRIMARY KEY,
 count integer NOT NULL,
 until_at timestamptz NOT NULL
);
CREATE INDEX rate_limits_expiry ON rate_limits(until_at);
ALTER TABLE mail_queue ADD COLUMN status text NOT NULL DEFAULT 'pending'
 CHECK (status IN ('pending','delivered','failed','legacy_finished'));
ALTER TABLE mail_queue ADD COLUMN error_code text;
UPDATE mail_queue SET status='legacy_finished' WHERE finished_at IS NOT NULL;
