# Structure

## 修改点

### 路由注册逻辑
1. 不要在 ws.Route(ws.PUT("/users/{userId}").To(rest.UpdateResource(scope, userStorage, validateUserUpdate, "userId"))) 每次路由注册的时候都传一遍scope，
对于REST API只需要初始化并传入一次（同时支持在 Container/WebService 创建时注入 Scope ，WebService的 Scope 优先），同时避免每次请求到达时创建并传入
2. 不要传入 "userId" 仅作为验证条件，我要直接在 validateUserUpdate 内自定义验证逻辑，所有资源 validate 逻辑都内聚到 Store 里
3. 支持子资源 API ，比如 /usernames/{usernameId}/users/{userId}
4. 不同资源的 API 路由类似 K8s 的方式自动注册（根据 http method）

### 模块化

1. 删除原有 pkg/modules 内容
2. 在根目录创建 pkg/apis 文件夹，在 pkg/apis 里分业务模块，所有业务代码（比如User、Namespace）的数据库操作、业务逻辑、路由拼装都在模块内完成，高内聚、易于维护。
仅需在 app/lcp-server/handler 中用少量代码注册每个模块的 API
3. 模块代码采用分层架构，支持类似 K8s 的 API 版本管理
4. 去除 Service 层，直接通过每个资源 Storage 来操作，实现高内聚
5. 支持子资源 API ，比如 /usernames/{usernameId}/users/{userId}
6. 移除现有 lib/db 里的代码，重新在 pkg/db 里创建 sqlc 配置，要求在 pkg/db 里定义 sql 表结构，在不同模块、资源下生成对应的数据库操作代码

### OpenAPI

1. 参考 K8s 的 REST API 文档生成功能，在 User、Namespace结构体字段以及路由注册函数中进行各种注释，
可以自动生成可以被 Swagger 等工具解析展示的 API 文档文件
2. 不要引入第三方 Swagger 生成库，直接在 lib/openapi 包里创建
3. 支持模块注释（转化为group）
4. 支持扫描 pkg/apis 下所有模块，自动生成 OpenAPI 文件

## 要点
1. 保证高内聚，易于维护，能在一定地方定义引入，就不要多次多个地方定义引入
2. 避免多次类型转化，能用一个结构体解决，就内聚到一个结构体中
3. 保持灵活性，易于扩展特定的业务逻辑，比如向 Namespace 添加用户，判断该 Namespace 里是否已存在该用户等等
