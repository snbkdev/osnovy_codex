ALTER TABLE users ADD COLUMN full_name text NOT NULL DEFAULT '' CHECK (char_length(full_name) <= 200);
ALTER TABLE users ADD COLUMN date_of_birth date CHECK (date_of_birth >= DATE '0001-01-01' AND isfinite(date_of_birth));
