ALTER TABLE refresh_tokens ADD COLUMN request_hash text;
ALTER TABLE refresh_tokens ADD COLUMN next_hash text;
