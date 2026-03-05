# 组织管理

## 背景

增加 Workspace 对象，代表租户，实现多租户管理的功能，一个 Workspace 下面可以有多个 Namespace （项目）

## 需求

### Workspace 管理

1. 常规 Workspace 的增删改查
2. 默认 Workspace （默认租户）存在于初始化数据

### Workspace 下 Namespace 管理

1. 每个 Workspace 创建后都包含一个默认 Namespace
2. Namespace 必须创建在某一 Workspace 下
3. 支持 /namespaces 查询所有 Workspace 的所有 Namespace ，也支持/workspaces/{workspaceId}/namespaces 查询该 Workspace 下所有的 Namespace 
4. 删除 Workspace 需判断是否该 Workspace 下是否存在 Namespace ，只有删除了所有子资源，才能删除子资源

### Workspace、Namespace 下 User 管理

1. 可以将 User 添加进 Workspace/Namespace，一个 User 能添加进多个 namespace 与 workspace ，添加进了一个 Namespace 即自动添加进 Namespace 上层对应的Workspace
2. 可以灵活查询不同组织的用户：/users (all users), /workspaces/{workspaceId}/users (all users in this workspace),
/workspaces/{workspaceId}/namespaces/{namespaceId}/users/{userId} (all users in this namespace)