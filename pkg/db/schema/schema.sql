-- users table
CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    username      VARCHAR(255) NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    display_name  VARCHAR(255) NOT NULL DEFAULT '',
    phone         VARCHAR(50)  NOT NULL UNIQUE,
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


-- permissions table (auto-generated from routes, read-only)
CREATE TABLE permissions (
    id          BIGSERIAL    PRIMARY KEY,
    code        VARCHAR(255) NOT NULL UNIQUE,
    method      VARCHAR(10)  NOT NULL,
    path        VARCHAR(512) NOT NULL,
    scope       VARCHAR(20)  NOT NULL DEFAULT 'platform',
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE permissions IS '权限表：从路由自动生成，系统只读';
COMMENT ON COLUMN permissions.code IS '权限标识，如 iam:users:list，首段为模块名';
COMMENT ON COLUMN permissions.method IS 'HTTP 方法，如 GET、POST';
COMMENT ON COLUMN permissions.path IS '规范 API 路径';
COMMENT ON COLUMN permissions.scope IS '权限作用域：platform, workspace, namespace';
COMMENT ON COLUMN permissions.description IS '权限描述';

-- roles table (builtin + user-defined)
CREATE TABLE roles (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    display_name  VARCHAR(255) NOT NULL DEFAULT '',
    description   TEXT         NOT NULL DEFAULT '',
    scope         VARCHAR(20)  NOT NULL,
    workspace_id  BIGINT       REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id  BIGINT       REFERENCES namespaces(id) ON DELETE CASCADE,
    builtin       BOOLEAN      NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_role_scope CHECK (
        (scope = 'platform'  AND workspace_id IS NULL     AND namespace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND namespace_id IS NULL) OR
        (scope = 'namespace' AND workspace_id IS NULL     AND namespace_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uk_roles_platform
    ON roles(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX uk_roles_workspace
    ON roles(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uk_roles_namespace
    ON roles(name, namespace_id) WHERE scope = 'namespace';

COMMENT ON TABLE roles IS '角色表：内置角色 + 用户自定义角色';
COMMENT ON COLUMN roles.name IS '角色名称，同 scope 内唯一';
COMMENT ON COLUMN roles.display_name IS '角色显示名称';
COMMENT ON COLUMN roles.scope IS '角色作用域：platform / workspace / namespace';
COMMENT ON COLUMN roles.workspace_id IS '所属租户 ID（workspace scope 时必填）';
COMMENT ON COLUMN roles.namespace_id IS '所属项目 ID（namespace scope 时必填）';
COMMENT ON COLUMN roles.builtin IS '是否为内置角色（内置不可删除）';

-- role permission rules (supports exact codes and wildcard patterns)
CREATE TABLE role_permission_rules (
    role_id  BIGINT       NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    pattern  VARCHAR(255) NOT NULL,
    PRIMARY KEY (role_id, pattern)
);

COMMENT ON TABLE role_permission_rules IS '角色权限规则：支持精确匹配和通配符模式';
COMMENT ON COLUMN role_permission_rules.role_id IS '关联角色 ID';
COMMENT ON COLUMN role_permission_rules.pattern IS '权限模式：*:*（全通配）、iam:*（前缀）、*:list（后缀）、iam:users:list（精确）';

-- role bindings (user + role + resource instance)
CREATE TABLE role_bindings (
    id            BIGSERIAL   PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id       BIGINT      NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope         VARCHAR(20) NOT NULL,
    workspace_id  BIGINT      REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id  BIGINT      REFERENCES namespaces(id) ON DELETE CASCADE,
    is_owner      BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_binding_scope CHECK (scope IN ('platform', 'workspace', 'namespace')),
    CONSTRAINT chk_binding_ids CHECK (
        (scope = 'platform'  AND workspace_id IS NULL AND namespace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND namespace_id IS NULL) OR
        (scope = 'namespace' AND namespace_id IS NOT NULL AND workspace_id IS NOT NULL)
    )
);

COMMENT ON TABLE role_bindings IS '角色绑定：用户与角色的关联，带具体资源实例';
COMMENT ON COLUMN role_bindings.user_id IS '绑定用户 ID';
COMMENT ON COLUMN role_bindings.role_id IS '绑定角色 ID';
COMMENT ON COLUMN role_bindings.scope IS '绑定作用域：platform / workspace / namespace';
COMMENT ON COLUMN role_bindings.workspace_id IS '租户 ID（workspace/namespace scope 时必填）';
COMMENT ON COLUMN role_bindings.namespace_id IS '项目 ID（namespace scope 时必填）';
COMMENT ON COLUMN role_bindings.is_owner IS '是否为资源所有者（ownership 转移时更新）';

CREATE UNIQUE INDEX uk_role_bindings_platform
    ON role_bindings(user_id, role_id) WHERE scope = 'platform';
CREATE UNIQUE INDEX uk_role_bindings_workspace
    ON role_bindings(user_id, role_id, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uk_role_bindings_namespace
    ON role_bindings(user_id, role_id, namespace_id) WHERE scope = 'namespace';

CREATE INDEX idx_role_bindings_user ON role_bindings(user_id);
CREATE INDEX idx_role_bindings_workspace ON role_bindings(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_role_bindings_namespace ON role_bindings(namespace_id) WHERE namespace_id IS NOT NULL;

-- audit_logs table (immutable records, no FK constraints)
CREATE TABLE audit_logs (
    id            BIGSERIAL    PRIMARY KEY,
    user_id       BIGINT,                          -- nullable (failed login for unknown user)
    username      VARCHAR(255) NOT NULL DEFAULT '', -- denormalized, no FK
    event_type    VARCHAR(50)  NOT NULL,            -- 'api_operation' | 'authentication'
    action        VARCHAR(50)  NOT NULL,            -- 'create'|'update'|'patch'|'delete'|'deleteCollection'|'login'|'login_failed'|'token_refresh'|'token_refresh_blocked'
    resource_type VARCHAR(100) NOT NULL DEFAULT '', -- 'users'|'workspaces:namespaces'|...
    resource_id   VARCHAR(100) NOT NULL DEFAULT '',
    module        VARCHAR(50)  NOT NULL DEFAULT '', -- 'iam'|'dashboard'|...
    scope         VARCHAR(20)  NOT NULL DEFAULT 'platform',
    workspace_id  BIGINT,
    namespace_id  BIGINT,
    http_method   VARCHAR(10)  NOT NULL DEFAULT '',
    http_path     VARCHAR(500) NOT NULL DEFAULT '',
    status_code   INT          NOT NULL DEFAULT 0,
    client_ip     VARCHAR(45)  NOT NULL DEFAULT '',
    user_agent    VARCHAR(500) NOT NULL DEFAULT '',
    duration_ms   INT          NOT NULL DEFAULT 0,
    success       BOOLEAN      NOT NULL DEFAULT true,
    detail        TEXT         NOT NULL DEFAULT '', -- JSON for extra context
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id       ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at    ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_event_type    ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_workspace_id  ON audit_logs(workspace_id);
CREATE INDEX idx_audit_logs_namespace_id  ON audit_logs(namespace_id);

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

-- audit_logs table (immutable records, no FK constraints)
CREATE TABLE audit_logs (
    id            BIGSERIAL    PRIMARY KEY,
    user_id       BIGINT,
    username      VARCHAR(255) NOT NULL DEFAULT '',
    event_type    VARCHAR(50)  NOT NULL,
    action        VARCHAR(50)  NOT NULL,
    resource_type VARCHAR(100) NOT NULL DEFAULT '',
    resource_id   VARCHAR(100) NOT NULL DEFAULT '',
    module        VARCHAR(50)  NOT NULL DEFAULT '',
    scope         VARCHAR(20)  NOT NULL DEFAULT 'platform',
    workspace_id  BIGINT,
    namespace_id  BIGINT,
    http_method   VARCHAR(10)  NOT NULL DEFAULT '',
    http_path     VARCHAR(500) NOT NULL DEFAULT '',
    status_code   INT          NOT NULL DEFAULT 0,
    client_ip     VARCHAR(45)  NOT NULL DEFAULT '',
    user_agent    VARCHAR(500) NOT NULL DEFAULT '',
    duration_ms   INT          NOT NULL DEFAULT 0,
    success       BOOLEAN      NOT NULL DEFAULT true,
    detail        JSONB,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id       ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at    ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_event_type    ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_workspace_id  ON audit_logs(workspace_id);
CREATE INDEX idx_audit_logs_namespace_id  ON audit_logs(namespace_id);
