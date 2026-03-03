# 项目结构重构设计文档

## 概述

本文档描述了 LCP 项目的结构重构设计，主要目标是：
1. 简化路由注册逻辑，避免重复传递 scope 和验证参数
2. 实现模块化架构，按业务模块组织代码，高内聚低耦合
3. 支持 OpenAPI 文档自动生成
4. 支持多 API 版本

## 设计原则

- **高内聚低耦合**: 每个业务模块独立，包含完整的类型、存储、验证、服务和路由
- **接口驱动**: 通过接口检查自动注册路由，无需手动配置
- **配置继承**: Scope 支持 Container 和 WebService 两级配置，优先级 Container > WebService
- **声明式注册**: 只需声明资源名称和 Storage，框架自动完成路由注册
- **多版本支持**: 参考 K8s，支持同一资源的多个 API 版本

## 目录结构

### 新的项目结构

```
lcp/
├── pkg/
│   └── apis/                           # API 资源模块
│       ├── user/                       # User 模块
│       │   ├── v1/                     # v1 版本
│       │   │   ├── types/              # 类型定义
│       │   │   │   ├── doc.go          # 包级别注解
│       │   │   │   └── user.go         # User 类型
│       │   │   ├── storage/            # Storage 实现
│       │   │   │   └── storage.go
│       │   │   └── routes.go           # 路由注册
│       │   └── v2/                     # v2 版本（未来）
│       │       └── ...
│       └── namespace/                  # Namespace 模块
│           └── v1/
│               └── ...
├── lib/
│   ├── rest/                           # REST 框架
│   │   ├── handler.go                  # 现有的 handler
│   │   ├── scope.go                    # 新增：Scope 管理
│   │   ├── registry.go                 # 新增：自动路由注册
│   │   └── storage.go                  # Storage 接口（添加 Validator）
│   ├── openapi/                        # 新增：OpenAPI 生成器
│   │   ├── generator.go                # 文档生成器
│   │   ├── parser.go                   # 代码解析器
│   │   ├── spec.go                     # OpenAPI 规范定义
│   │   ├── annotations.go              # 注解解析
│   │   └── writer.go                   # 文档输出
│   └── ...
└── app/
    └── lcp-server/
        └── handler/
            └── handler.go              # 简化的注册逻辑
```

## 核心设计

### 1. Scope 管理机制

#### Container 和 WebService 支持 Scope

```go
// lib/rest/scope.go

// Container 添加 scope 支持
type Container struct {
    // ... 现有字段
    defaultScope *RequestScope  // 新增：默认 scope
}

func (c *Container) WithScope(scope *RequestScope) *Container {
    c.defaultScope = scope
    return c
}

// WebService 添加 scope 支持
type WebService struct {
    // ... 现有字段
    scope *RequestScope  // 新增：WebService 级别的 scope
}

func (ws *WebService) WithScope(scope *RequestScope) *WebService {
    ws.scope = scope
    return ws
}
```

#### Scope 通过闭包绑定到 Handler

在注册路由时，通过闭包将 scope 绑定到每个 handler 函数：

```go
func InstallAPIGroup(container *Container, info *APIGroupInfo) error {
    ws := new(WebService).Path(info.BasePath())

    // 在注册时就确定有效的 scope（优先级：Container > WebService）
    scope := container.defaultScope
    if scope == nil {
        scope = ws.scope
    }
    if scope == nil {
        return fmt.Errorf("no scope configured")
    }

    // 创建 handler 时将 scope 通过闭包绑定
    for resourceName, storage := range info.Resources {
        if creater, ok := storage.(Creater); ok {
            ws.Route(ws.POST("/"+resourceName).
                To(createResourceHandler(scope, storage)))
        }
        // ... 其他路由
    }

    container.Add(ws)
    return nil
}
```

### 2. 自动路由注册机制

#### APIGroupInfo 结构

```go
// lib/rest/registry.go

type APIGroupInfo struct {
    GroupName    string                      // API 组名，如 "core.lcp.io"
    GroupVersion string                      // 版本，如 "v1"
    Resources    map[string]rest.Storage     // 资源名 -> Storage
    SubResources []*SubResourceInfo          // 子资源定义
}

type SubResourceInfo struct {
    ParentResource string           // 父资源名，如 "namespaces"
    SubResource    string           // 子资源名，如 "users"
    Storage        rest.Storage     // 子资源的 Storage
}

// BasePath 自动生成
func (info *APIGroupInfo) BasePath() string {
    // core group 使用 /api/{version}
    if info.GroupName == "" || info.GroupName == "core" {
        return fmt.Sprintf("/api/%s", info.GroupVersion)
    }
    // 其他 group 使用 /apis/{group}/{version}
    return fmt.Sprintf("/apis/%s/%s", info.GroupName, info.GroupVersion)
}
```

#### 基于接口的自动注册

```go
func InstallAPIGroup(container *Container, info *APIGroupInfo) error {
    ws := new(WebService).Path(info.BasePath())
    scope := // ... 获取 scope

    // 注册顶级资源
    for resourceName, storage := range info.Resources {
        validator, _ := storage.(Validator)

        // 检查接口并自动注册路由
        if _, ok := storage.(Creater); ok {
            var validateFunc ValidateObjectFunc
            if validator != nil {
                validateFunc = validator.ValidateCreate
            }
            ws.Route(ws.POST("/"+resourceName).
                To(createResourceHandler(scope, storage, validateFunc)))
        }

        if _, ok := storage.(Getter); ok {
            ws.Route(ws.GET("/"+resourceName+"/{id}").
                To(getResourceHandler(scope, storage)))
        }

        // ... 其他接口检查
    }

    // 注册子资源
    for _, subRes := range info.SubResources {
        basePath := "/" + subRes.ParentResource + "/{" + subRes.ParentResource + "Id}/" + subRes.SubResource
        installResource(ws, scope, basePath, subRes.Storage)
    }

    container.Add(ws)
    return nil
}
```

### 3. Storage 内置验证

#### Validator 接口

```go
// lib/rest/storage.go

// Validator 接口 - Storage 可选实现
type Validator interface {
    ValidateCreate(ctx context.Context, obj runtime.Object) error
    ValidateUpdate(ctx context.Context, obj runtime.Object) error
    ValidatePatch(ctx context.Context, obj runtime.Object) error
    ValidateDelete(ctx context.Context, obj runtime.Object) error
}
```

验证逻辑完全内聚在 Storage 中，无需在 APIGroupInfo 中单独配置。

### 4. 模块化结构

#### User 模块示例

**pkg/apis/user/v1/types/doc.go**
```go
// +openapi:gen=true
// +openapi:groupName=core.lcp.io
// +openapi:groupVersion=v1

// Package v1 contains API types for user resources version 1.
package v1
```

**pkg/apis/user/v1/types/user.go**
```go
// User is the API representation of a user resource.
// +openapi:gen=true
// +openapi:resource=users
type User struct {
    runtime.TypeMeta `json:",inline"`

    // Standard object metadata.
    ObjectMeta `json:"metadata"`

    // Specification of the desired user.
    Spec UserSpec `json:"spec"`
}

// UserSpec holds user-specific fields.
// +openapi:gen=true
type UserSpec struct {
    // Username is the unique identifier for the user.
    // +openapi:required
    // +openapi:minLength=3
    // +openapi:maxLength=50
    Username string `json:"username"`

    // Email address of the user.
    // +openapi:required
    // +openapi:format=email
    Email string `json:"email"`

    // Display name of the user.
    // +openapi:maxLength=100
    DisplayName string `json:"displayName,omitempty"`
}
```

**pkg/apis/user/v1/storage/storage.go**
```go
package storage

type Storage struct {
    svc *service.Service
}

func New(svc *service.Service) rest.Storage {
    return &Storage{svc: svc}
}

// 实现 CRUD 接口
func (s *Storage) Get(ctx context.Context, id string) (runtime.Object, error) {
    return s.svc.Users().GetUser(ctx, id)
}

func (s *Storage) Create(ctx context.Context, obj runtime.Object, validate rest.ValidateObjectFunc, options *rest.CreateOptions) (runtime.Object, error) {
    // ... 实现
}

// 实现 Validator 接口
func (s *Storage) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    user, ok := obj.(*types.User)
    if !ok {
        return apierrors.NewBadRequest("invalid object type", nil)
    }

    if errs := validation.ValidateUserCreate(&user.Spec); errs.HasErrors() {
        return apierrors.NewBadRequest("validation failed", errs)
    }
    return nil
}

// ... 其他验证方法
```

**pkg/apis/user/v1/routes.go**
```go
package v1

import (
    "lcp.io/lcp/pkg/apis/user/v1/storage"
)

// NewStorage 创建 User 资源的 Storage
func NewStorage(svc *service.Service) rest.Storage {
    return storage.New(svc)
}
```

### 5. 注册方式

#### 简化的注册代码

```go
// app/lcp-server/handler/handler.go

func (a *APIServerHandler) InstallAPIs() error {
    logger.Infof("installing lcp-server APIs...")

    // 设置 Container 级别的 scope（所有 API 共享）
    scope := &rest.RequestScope{Serializer: runtime.NewCodecFactory()}
    a.GoRestfulContainer.WithScope(scope)

    // 批量注册多个 API Group 和版本
    groups := []*rest.APIGroupInfo{
        // Core API Group - v1
        {
            GroupName:    "core.lcp.io",
            GroupVersion: "v1",
            Resources: map[string]rest.Storage{
                "users":      userv1.NewStorage(a.svc),
                "namespaces": namespacev1.NewStorage(a.svc),
            },
            SubResources: []*rest.SubResourceInfo{
                {
                    ParentResource: "namespaces",
                    SubResource:    "users",
                    Storage:        namespaceuserv1.NewStorage(a.svc),
                },
            },
        },
        // Core API Group - v2
        {
            GroupName:    "core.lcp.io",
            GroupVersion: "v2",
            Resources: map[string]rest.Storage{
                "users": userv2.NewStorage(a.svc),
            },
        },
        // Apps API Group - v1
        {
            GroupName:    "apps.lcp.io",
            GroupVersion: "v1",
            Resources: map[string]rest.Storage{
                "deployments": deploymentv1.NewStorage(a.svc),
            },
        },
    }

    return rest.InstallAPIGroups(a.GoRestfulContainer, groups...)
}
```

#### 生成的路由示例

```
# Core API Group
/api/v1/users
/api/v1/users/{id}
/api/v1/namespaces
/api/v1/namespaces/{id}
/api/v1/namespaces/{namespaceId}/users
/api/v1/namespaces/{namespaceId}/users/{userId}

/api/v2/users
/api/v2/users/{id}

# Apps API Group
/apis/apps.lcp.io/v1/deployments
/apis/apps.lcp.io/v1/deployments/{id}
```

### 6. OpenAPI 文档生成

#### 注解标记

**包级别（doc.go）：**
```
+openapi:gen=true                    # 为整个包生成 OpenAPI 定义
+openapi:groupName=<name>            # API 组名，如 core.lcp.io
+openapi:groupVersion=<version>      # API 版本，如 v1
```

**类型级别：**
```
+openapi:gen=true                    # 生成此类型的 OpenAPI 定义
+openapi:resource=<name>             # 资源名称（用于路径）
```

**字段级别：**
```
+openapi:required                    # 必填字段
+openapi:minLength=<n>               # 字符串最小长度
+openapi:maxLength=<n>               # 字符串最大长度
+openapi:pattern=<regex>             # 正则表达式验证
+openapi:format=<format>             # 格式（email, date, uuid, uri 等）
+openapi:minimum=<n>                 # 数值最小值
+openapi:maximum=<n>                 # 数值最大值
+openapi:enum=<v1,v2,v3>             # 枚举值（逗号分隔）
+openapi:default=<value>             # 默认值
```

#### 生成器使用

```go
func generateOpenAPISpec() {
    gen := openapi.NewGenerator(&openapi.Config{
        Title:       "LCP API",
        Description: "LCP Platform API Documentation",
        AutoScan:    true,  // 自动扫描 pkg/apis 目录
    })

    spec, err := gen.Generate()
    if err != nil {
        log.Fatal(err)
    }

    // 输出 JSON 和 YAML 格式
    openapi.WriteJSON(spec, "docs/openapi.json")
    openapi.WriteYAML(spec, "docs/openapi.yaml")
}
```

生成器会自动：
1. 递归扫描 `pkg/apis/` 目录
2. 识别所有带 `+openapi:gen=true` 的包
3. 解析类型定义和注解
4. 生成完整的 OpenAPI 3.0 规范文档

## 实施计划

### 阶段一：基础框架改造（1-2天）

1. **扩展 REST 框架**
   - 在 `lib/rest/scope.go` 中实现 Scope 继承机制
   - 在 `lib/rest/registry.go` 中实现自动路由注册
   - 修改 handler 函数，通过闭包绑定 scope
   - 添加 Validator 接口到 `lib/rest/storage.go`

2. **创建 OpenAPI 生成器骨架**
   - 创建 `lib/openapi/` 目录结构
   - 实现基础的代码解析器（使用 go/ast）
   - 实现注解解析逻辑
   - 实现 OpenAPI 3.0 规范输出

### 阶段二：User 模块迁移（1天）

3. **创建新的模块结构**
   - 创建 `pkg/apis/user/v1/` 目录结构
   - 迁移类型定义到 `pkg/apis/user/v1/types/`
   - 迁移 Storage 实现到 `pkg/apis/user/v1/storage/`
   - 在 Storage 中实现 Validator 接口
   - 创建 `pkg/apis/user/v1/routes.go`

4. **添加 OpenAPI 注解**
   - 创建 `pkg/apis/user/v1/types/doc.go`
   - 为 User 类型添加注解
   - 为字段添加验证注解

5. **更新注册逻辑**
   - 修改 `app/lcp-server/handler/handler.go`
   - 使用新的 APIGroupInfo 方式注册
   - 测试 User API 功能

### 阶段三：验证和完善（0.5天）

6. **测试验证**
   - 测试所有 User API 端点
   - 验证验证逻辑正常工作
   - 生成 OpenAPI 文档并验证

7. **文档和示例**
   - 编写模块开发指南
   - 提供 Namespace 模块迁移示例

### 阶段四：其他模块迁移（按需）

8. **迁移 Namespace 模块**
   - 按照 User 模块的模式迁移
   - 如果有子资源，使用 SubResourceInfo

9. **清理旧代码**
   - 删除 `app/lcp-server/handler/` 下的旧文件
   - 更新导入路径

## 关键文件清单

### 新增文件

```
lib/rest/scope.go                      # Scope 管理
lib/rest/registry.go                   # 自动注册
lib/openapi/generator.go               # OpenAPI 生成器
lib/openapi/parser.go                  # 代码解析
lib/openapi/spec.go                    # 规范定义
lib/openapi/annotations.go             # 注解解析
lib/openapi/writer.go                  # 文档输出
pkg/apis/user/v1/types/doc.go          # 包级别注解
pkg/apis/user/v1/types/user.go         # 类型定义
pkg/apis/user/v1/storage/storage.go    # Storage 实现
pkg/apis/user/v1/routes.go             # 注册函数
```

### 修改文件

```
lib/rest/handler.go                    # 简化 handler 函数签名
lib/rest/storage.go                    # 添加 Validator 接口
lib/rest/webservice.go                 # 添加 WithScope 方法
lib/rest/container.go                  # 添加 WithScope 方法
app/lcp-server/handler/handler.go      # 简化注册逻辑
```

### 删除文件（阶段四）

```
app/lcp-server/handler/user_storage.go
app/lcp-server/handler/validation.go
```

## 风险控制

1. **向后兼容**：在迁移完成前，新旧代码可以共存
2. **独立测试**：每个模块迁移后独立测试
3. **回滚方案**：保留旧代码直到新代码完全验证
4. **文档先行**：先完善开发文档，再迁移其他模块

## 总结

本次重构将带来以下改进：

1. **简化路由注册**：从每个路由都需要传递 scope 和验证参数，简化为只需声明资源和 Storage
2. **高内聚模块**：每个业务模块独立完整，包含类型、存储、验证和路由
3. **自动化文档**：通过注解自动生成 OpenAPI 文档，无需手动维护
4. **多版本支持**：参考 K8s，支持同一资源的多个 API 版本
5. **易于扩展**：新增模块只需创建目录和实现接口，零配置自动注册

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
