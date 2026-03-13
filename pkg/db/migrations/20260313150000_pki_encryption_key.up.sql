CREATE TABLE pki_encryption_keys (
    id              BIGSERIAL PRIMARY KEY,
    encryption_key  BYTEA NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE pki_encryption_keys IS 'PKI 加密密钥：AES-256 密钥，用于加密证书私钥';
