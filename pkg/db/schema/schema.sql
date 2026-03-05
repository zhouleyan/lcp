-- users table
CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    username      VARCHAR(255) NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    display_name  VARCHAR(255) NOT NULL DEFAULT '',
    phone         VARCHAR(50)  NOT NULL DEFAULT '',
    avatar_url    VARCHAR(512) NOT NULL DEFAULT '',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_display_name ON users(display_name);

-- workspaces table
CREATE TABLE workspaces (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    owner_id     BIGINT       NOT NULL REFERENCES users(id),
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_workspaces_owner_id ON workspaces(owner_id);
CREATE INDEX idx_workspaces_status ON workspaces(status);
CREATE INDEX idx_workspaces_created_at ON workspaces(created_at);

-- namespaces table
CREATE TABLE namespaces (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    workspace_id BIGINT       NOT NULL REFERENCES workspaces(id),
    owner_id     BIGINT       NOT NULL REFERENCES users(id),
    visibility   VARCHAR(20)  NOT NULL DEFAULT 'private',
    max_members  INT          NOT NULL DEFAULT 0,
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_namespaces_workspace_id ON namespaces(workspace_id);
CREATE INDEX idx_namespaces_owner_id ON namespaces(owner_id);
CREATE INDEX idx_namespaces_status ON namespaces(status);
CREATE INDEX idx_namespaces_visibility ON namespaces(visibility);
CREATE INDEX idx_namespaces_created_at ON namespaces(created_at);

-- user_workspaces join table (many-to-many)
CREATE TABLE user_workspaces (
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT      NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role         VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, workspace_id)
);

CREATE INDEX idx_user_workspaces_workspace_id ON user_workspaces(workspace_id);
CREATE INDEX idx_user_workspaces_role ON user_workspaces(role);

-- user_namespaces join table (many-to-many)
CREATE TABLE user_namespaces (
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    namespace_id BIGINT      NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    role         VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, namespace_id)
);

CREATE INDEX idx_user_namespaces_namespace_id ON user_namespaces(namespace_id);
CREATE INDEX idx_user_namespaces_role ON user_namespaces(role);

-- refresh_tokens table
CREATE TABLE refresh_tokens (
    id         BIGSERIAL    PRIMARY KEY,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id  VARCHAR(255) NOT NULL,
    scope      TEXT         NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked    BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
