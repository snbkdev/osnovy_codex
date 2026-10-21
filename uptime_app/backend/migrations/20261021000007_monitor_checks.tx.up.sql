ALTER TABLE monitors
 ADD COLUMN status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','success','error')),
 ADD COLUMN http_status integer NOT NULL DEFAULT 0,
 ADD COLUMN check_error text NOT NULL DEFAULT '',
 ADD COLUMN checked_at timestamptz,
 ADD COLUMN next_check_at timestamptz NOT NULL DEFAULT now(),
 ADD COLUMN check_token text NOT NULL DEFAULT '';
CREATE INDEX monitors_next_check ON monitors(next_check_at,id);
