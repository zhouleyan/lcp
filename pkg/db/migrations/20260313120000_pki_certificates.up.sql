CREATE TABLE certificates (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) UNIQUE NOT NULL,
    cert_type       VARCHAR(16) NOT NULL,
    common_name     VARCHAR(256) NOT NULL DEFAULT '',
    dns_names       TEXT[] NOT NULL DEFAULT '{}',
    ca_name         VARCHAR(128),
    serial_number   VARCHAR(64) NOT NULL,
    certificate     BYTEA NOT NULL,
    private_key     BYTEA NOT NULL,
    not_before      TIMESTAMPTZ NOT NULL,
    not_after       TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_certificates_cert_type ON certificates(cert_type);
CREATE INDEX idx_certificates_ca_name ON certificates(ca_name);
