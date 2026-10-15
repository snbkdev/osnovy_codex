CREATE TABLE monitors (
 id text PRIMARY KEY,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 url text NOT NULL CHECK (char_length(url) BETWEEN 1 AND 2048),
 interval_seconds integer NOT NULL CHECK (interval_seconds BETWEEN 5 AND 604800),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX monitors_user_created ON monitors(user_id, created_at DESC, id DESC);
