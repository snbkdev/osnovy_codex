CREATE TABLE monitor_checks (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 monitor_id text NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
 url text NOT NULL,
 checked_at timestamptz NOT NULL,
 status text NOT NULL CHECK (status IN ('success','error')),
 http_status integer NOT NULL,
 check_error text NOT NULL DEFAULT ''
);
CREATE INDEX monitor_checks_monitor_time ON monitor_checks(monitor_id,checked_at);
-- Preserve the one genuine sample already available for existing monitors.
INSERT INTO monitor_checks(monitor_id,url,checked_at,status,http_status,check_error)
 SELECT id,url,checked_at,status,http_status,check_error FROM monitors
 WHERE checked_at IS NOT NULL AND status IN ('success','error');
