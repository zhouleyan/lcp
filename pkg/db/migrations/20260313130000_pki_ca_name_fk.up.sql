ALTER TABLE certificates
    ADD CONSTRAINT fk_certificates_ca_name
    FOREIGN KEY (ca_name) REFERENCES certificates(name);
