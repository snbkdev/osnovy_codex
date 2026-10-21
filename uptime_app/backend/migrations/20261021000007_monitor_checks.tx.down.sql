DROP INDEX monitors_next_check;
ALTER TABLE monitors DROP COLUMN check_token, DROP COLUMN next_check_at,
 DROP COLUMN checked_at, DROP COLUMN check_error, DROP COLUMN http_status, DROP COLUMN status;
