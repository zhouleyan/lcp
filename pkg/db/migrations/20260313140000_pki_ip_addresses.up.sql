ALTER TABLE certificates ADD COLUMN ip_addresses TEXT[] NOT NULL DEFAULT '{}';
