CREATE TABLE users (
 id text PRIMARY KEY,
 email text NOT NULL UNIQUE CHECK (email = lower(btrim(email))),
 password_hash text NOT NULL,
 email_verified boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
 id text PRIMARY KEY,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK (expires_at > created_at)
);
CREATE INDEX sessions_user_active ON sessions(user_id, expires_at) WHERE revoked_at IS NULL;
CREATE TABLE refresh_tokens (
 hash text PRIMARY KEY,
 session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
 used_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_session ON refresh_tokens(session_id);
CREATE TABLE one_time_tokens (
 hash text PRIMARY KEY,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 purpose text NOT NULL CHECK (purpose IN ('verify', 'reset')),
 expires_at timestamptz NOT NULL,
 UNIQUE (user_id, purpose)
);
CREATE INDEX one_time_tokens_expiry ON one_time_tokens(expires_at);
CREATE TABLE mail_queue (
 id bigserial PRIMARY KEY,
 payload bytea,
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
 available_at timestamptz NOT NULL DEFAULT now(),
 finished_at timestamptz,
 CHECK ((finished_at IS NULL AND payload IS NOT NULL) OR (finished_at IS NOT NULL AND payload IS NULL))
);
CREATE INDEX mail_queue_pending ON mail_queue(available_at) WHERE finished_at IS NULL;
