CREATE TABLE o11y_endpoints (
    id          BIGSERIAL    PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(500) NOT NULL DEFAULT '',
    public      BOOLEAN      NOT NULL DEFAULT true,
    metrics_url VARCHAR(500) NOT NULL DEFAULT '',
    logs_url    VARCHAR(500) NOT NULL DEFAULT '',
    traces_url  VARCHAR(500) NOT NULL DEFAULT '',
    apm_url     VARCHAR(500) NOT NULL DEFAULT '',
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_o11y_endpoints_status ON o11y_endpoints(status);
CREATE INDEX idx_o11y_endpoints_public ON o11y_endpoints(public);
