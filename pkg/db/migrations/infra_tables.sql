-- Infra module: incremental migration
-- Creates environments, hosts, host_assignments tables
-- Safe to run on existing database (uses IF NOT EXISTS / DROP IF EXISTS)

BEGIN;

-- ============================================================
-- 1. environments
-- ============================================================
CREATE TABLE IF NOT EXISTS environments (
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

CREATE UNIQUE INDEX IF NOT EXISTS uk_environments_platform  ON environments(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX IF NOT EXISTS uk_environments_workspace ON environments(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX IF NOT EXISTS uk_environments_namespace ON environments(name, namespace_id) WHERE scope = 'namespace';

CREATE INDEX IF NOT EXISTS idx_environments_scope        ON environments(scope);
CREATE INDEX IF NOT EXISTS idx_environments_workspace_id ON environments(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_environments_namespace_id ON environments(namespace_id) WHERE namespace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_environments_status       ON environments(status);
CREATE INDEX IF NOT EXISTS idx_environments_created_at   ON environments(created_at);

COMMENT ON TABLE environments IS '环境表：管理维度，用于按生命周期阶段分组资源';
COMMENT ON COLUMN environments.env_type IS '环境类型：development, testing, staging, production, custom';
COMMENT ON COLUMN environments.scope IS '作用域：platform / workspace / namespace';

-- ============================================================
-- 2. hosts
-- ============================================================
CREATE TABLE IF NOT EXISTS hosts (
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

CREATE UNIQUE INDEX IF NOT EXISTS uk_hosts_platform  ON hosts(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX IF NOT EXISTS uk_hosts_workspace ON hosts(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX IF NOT EXISTS uk_hosts_namespace ON hosts(name, namespace_id) WHERE scope = 'namespace';

CREATE INDEX IF NOT EXISTS idx_hosts_scope          ON hosts(scope);
CREATE INDEX IF NOT EXISTS idx_hosts_workspace_id   ON hosts(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_hosts_namespace_id   ON hosts(namespace_id) WHERE namespace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_hosts_environment_id ON hosts(environment_id) WHERE environment_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_hosts_status         ON hosts(status);
CREATE INDEX IF NOT EXISTS idx_hosts_created_at     ON hosts(created_at);
CREATE INDEX IF NOT EXISTS idx_hosts_labels         ON hosts USING GIN(labels);

COMMENT ON TABLE hosts IS '主机表：物理机或虚拟机资源';
COMMENT ON COLUMN hosts.scope IS '创建层级：platform / workspace / namespace';
COMMENT ON COLUMN hosts.environment_id IS '所属环境（一对一，ON DELETE SET NULL）';
COMMENT ON COLUMN hosts.labels IS '标签（JSON 对象，支持 GIN 索引查询）';

-- ============================================================
-- 3. host_assignments
-- ============================================================
CREATE TABLE IF NOT EXISTS host_assignments (
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

CREATE UNIQUE INDEX IF NOT EXISTS uk_host_assignment_workspace ON host_assignments(host_id, workspace_id) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_host_assignment_namespace  ON host_assignments(host_id, namespace_id) WHERE namespace_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_host_assignments_host      ON host_assignments(host_id);
CREATE INDEX IF NOT EXISTS idx_host_assignments_workspace ON host_assignments(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_host_assignments_namespace ON host_assignments(namespace_id) WHERE namespace_id IS NOT NULL;

COMMENT ON TABLE host_assignments IS '主机分配表：引用语义，上层主机授权给下层使用';
COMMENT ON COLUMN host_assignments.host_id IS '被分配的主机 ID';
COMMENT ON COLUMN host_assignments.workspace_id IS '目标租户（平台主机 → 租户）';
COMMENT ON COLUMN host_assignments.namespace_id IS '目标项目（平台/租户主机 → 项目）';

COMMIT;
