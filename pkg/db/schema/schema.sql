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
    code        VARCHAR(255) NOT NULL,
    method      VARCHAR(10)  NOT NULL,
    path        VARCHAR(512) NOT NULL,
    scope       VARCHAR(20)  NOT NULL DEFAULT 'platform',
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (code, scope)
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

-- environments table
CREATE TABLE environments (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    display_name  VARCHAR(255) NOT NULL DEFAULT '',
    description   TEXT         NOT NULL DEFAULT '',
    env_type      VARCHAR(50)  NOT NULL DEFAULT 'custom',
    scope         VARCHAR(20)  NOT NULL,
    workspace_id  BIGINT       REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id  BIGINT       REFERENCES namespaces(id) ON DELETE CASCADE,
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_env_scope CHECK (
        (scope = 'platform'  AND workspace_id IS NULL     AND namespace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND namespace_id IS NULL) OR
        (scope = 'namespace' AND workspace_id IS NULL     AND namespace_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uk_environments_platform  ON environments(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX uk_environments_workspace ON environments(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uk_environments_namespace ON environments(name, namespace_id) WHERE scope = 'namespace';

CREATE INDEX idx_environments_scope        ON environments(scope);
CREATE INDEX idx_environments_workspace_id ON environments(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_environments_namespace_id ON environments(namespace_id) WHERE namespace_id IS NOT NULL;
CREATE INDEX idx_environments_status       ON environments(status);
CREATE INDEX idx_environments_created_at   ON environments(created_at);

COMMENT ON TABLE environments IS '环境表：管理维度，用于按生命周期阶段分组资源';
COMMENT ON COLUMN environments.env_type IS '环境类型：development, testing, staging, production, custom';
COMMENT ON COLUMN environments.scope IS '作用域：platform / workspace / namespace';

-- hosts table
CREATE TABLE hosts (
    id              BIGSERIAL    PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    display_name    VARCHAR(255) NOT NULL DEFAULT '',
    description     TEXT         NOT NULL DEFAULT '',
    hostname        VARCHAR(255) NOT NULL DEFAULT '',
    ip_address      VARCHAR(45)  NOT NULL DEFAULT '',
    os              VARCHAR(100) NOT NULL DEFAULT '',
    arch            VARCHAR(50)  NOT NULL DEFAULT '',
    cpu_cores       INT          NOT NULL DEFAULT 0,
    memory_mb       BIGINT       NOT NULL DEFAULT 0,
    disk_gb         BIGINT       NOT NULL DEFAULT 0,
    labels          JSONB        NOT NULL DEFAULT '{}',
    scope           VARCHAR(20)  NOT NULL,
    workspace_id    BIGINT       REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id    BIGINT       REFERENCES namespaces(id) ON DELETE CASCADE,
    environment_id  BIGINT       REFERENCES environments(id) ON DELETE SET NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_host_scope CHECK (
        (scope = 'platform'  AND workspace_id IS NULL     AND namespace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND namespace_id IS NULL) OR
        (scope = 'namespace' AND workspace_id IS NULL     AND namespace_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uk_hosts_platform  ON hosts(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX uk_hosts_workspace ON hosts(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uk_hosts_namespace ON hosts(name, namespace_id) WHERE scope = 'namespace';

CREATE INDEX idx_hosts_scope          ON hosts(scope);
CREATE INDEX idx_hosts_workspace_id   ON hosts(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_hosts_namespace_id   ON hosts(namespace_id) WHERE namespace_id IS NOT NULL;
CREATE INDEX idx_hosts_environment_id ON hosts(environment_id) WHERE environment_id IS NOT NULL;
CREATE INDEX idx_hosts_status         ON hosts(status);
CREATE INDEX idx_hosts_created_at     ON hosts(created_at);
CREATE INDEX idx_hosts_labels         ON hosts USING GIN(labels);

COMMENT ON TABLE hosts IS '主机表：物理机或虚拟机资源';
COMMENT ON COLUMN hosts.scope IS '创建层级：platform / workspace / namespace';
COMMENT ON COLUMN hosts.environment_id IS '所属环境（一对一，ON DELETE SET NULL）';
COMMENT ON COLUMN hosts.labels IS '标签（JSON 对象，支持 GIN 索引查询）';

-- host_assignments table
CREATE TABLE host_assignments (
    id            BIGSERIAL   PRIMARY KEY,
    host_id       BIGINT      NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    workspace_id  BIGINT      REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id  BIGINT      REFERENCES namespaces(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_assignment_target CHECK (
        (workspace_id IS NOT NULL AND namespace_id IS NULL) OR
        (workspace_id IS NULL     AND namespace_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uk_host_assignment_workspace ON host_assignments(host_id, workspace_id) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX uk_host_assignment_namespace  ON host_assignments(host_id, namespace_id) WHERE namespace_id IS NOT NULL;

CREATE INDEX idx_host_assignments_host      ON host_assignments(host_id);
CREATE INDEX idx_host_assignments_workspace ON host_assignments(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_host_assignments_namespace ON host_assignments(namespace_id) WHERE namespace_id IS NOT NULL;

COMMENT ON TABLE host_assignments IS '主机分配表：引用语义，上层主机授权给下层使用';
COMMENT ON COLUMN host_assignments.host_id IS '被分配的主机 ID';
COMMENT ON COLUMN host_assignments.workspace_id IS '目标租户（平台主机 → 租户）';
COMMENT ON COLUMN host_assignments.namespace_id IS '目标项目（平台/租户主机 → 项目）';

-- oidc_keys table (auto-generated signing keys)
CREATE TABLE oidc_keys (
    id          BIGSERIAL    PRIMARY KEY,
    key_id      VARCHAR(64)  NOT NULL UNIQUE,
    private_key BYTEA        NOT NULL,
    public_key  BYTEA        NOT NULL,
    algorithm   VARCHAR(16)  NOT NULL DEFAULT 'EdDSA',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE oidc_keys IS 'OIDC 签名密钥：自动生成，存储 PEM 编码的密钥对';
COMMENT ON COLUMN oidc_keys.key_id IS 'RFC 7638 thumbprint，用于 JWK kid 字段';
COMMENT ON COLUMN oidc_keys.algorithm IS '签名算法：EdDSA, ES256, RS256';

-- networks table (logical VPC containers)
CREATE TABLE networks (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    cidr         VARCHAR(50)  NOT NULL DEFAULT '',
    max_subnets  INT          NOT NULL DEFAULT 10,
    is_public    BOOLEAN      NOT NULL DEFAULT true,
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_networks_status     ON networks(status);
CREATE INDEX idx_networks_created_at ON networks(created_at);

COMMENT ON TABLE networks IS '网络表：平台级 VPC 逻辑分组容器';
COMMENT ON COLUMN networks.name IS '网络名称，全局唯一';
COMMENT ON COLUMN networks.cidr IS '网络 CIDR 地址段（可选），限制子网 CIDR 分配范围';
COMMENT ON COLUMN networks.status IS '状态：active / inactive';

-- subnets table (CIDR + bitmap)
CREATE TABLE subnets (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    network_id   BIGINT       NOT NULL REFERENCES networks(id),
    cidr         VARCHAR(50)  NOT NULL,
    gateway      VARCHAR(45)  NOT NULL DEFAULT '',
    bitmap       BYTEA        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uk_subnet_network_name UNIQUE (network_id, name),
    CONSTRAINT uk_subnet_network_cidr UNIQUE (network_id, cidr)
);

CREATE INDEX idx_subnets_network_id ON subnets(network_id);
CREATE INDEX idx_subnets_created_at ON subnets(created_at);

COMMENT ON TABLE subnets IS '子网表：每个子网 = 一个 CIDR + 一个 bitmap';
COMMENT ON COLUMN subnets.network_id IS '所属网络 ID';
COMMENT ON COLUMN subnets.cidr IS 'CIDR 表示，如 10.0.0.0/24';
COMMENT ON COLUMN subnets.gateway IS '网关 IP 地址';
COMMENT ON COLUMN subnets.bitmap IS 'IP 分配位图（BYTEA）';

-- ip_allocations table (IP allocation records)
CREATE TABLE ip_allocations (
    id          BIGSERIAL    PRIMARY KEY,
    subnet_id   BIGINT       NOT NULL REFERENCES subnets(id),
    ip          VARCHAR(45)  NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    is_gateway  BOOLEAN      NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uk_allocation_subnet_ip UNIQUE (subnet_id, ip)
);

CREATE INDEX idx_ip_allocations_subnet_id ON ip_allocations(subnet_id);

COMMENT ON TABLE ip_allocations IS 'IP 分配记录表：记录每个已分配的 IP';
COMMENT ON COLUMN ip_allocations.subnet_id IS '所属子网 ID';
COMMENT ON COLUMN ip_allocations.ip IS '分配的 IP 地址';
COMMENT ON COLUMN ip_allocations.is_gateway IS '是否为网关地址（自动分配，不可手动删除）';
