# lcp
LCP

## Start LCP Server

```bash
lcp-server \
-httpListenerAddr=:8421 \
-httpListenerAddr=:8422 \
-httpListenerAddr=:8423
```

## TODO

### REST API Routes

http.Server.Handler -> APIServerHandler -> goRestfulContainer := restful.NewContainer(...)

APIServerHandler.ServeHTTP(...) -> FullHandlerChain.ServeHTTP(...) -> director.ServeHTTP(...) -> goRestfulContainer.Dispatch(w, req) -> c.dispatch(w, req)

Request Dispatch
```txt
HTTP Request
    ↓
FullHandlerChain.ServeHTTP (包含各种过滤器)
    ↓
director.ServeHTTP
    ↓
goRestfulContainer.Dispatch (直接调用，跳过 ServeHTTP)
    ↓
Router.SelectRoute
    ↓
ExtractParameters
    ↓
FilterChain.ProcessFilter
    ↓
Route.Function
```

Route register
```txt
GenericAPIServer.InstallAPIGroups
    ↓
installAPIResources
    ↓
APIGroupVersion.InstallREST
    ↓
APIInstaller.Install
    ↓
newWebService
    ↓
ws.Path("/apis/apps/v1")
    ↓
registerResourceHandlers("pods", storage, ws)
    ↓
container.Add(ws)
```

### Consumes、Produces 定义

| 概念           | 位置             | 含义                 |
|--------------|----------------|--------------------|
| Consumes     | Route 定义       | 路由可消费（接收）的 MIME 类型 |
| Produces     | Route 定义       | 路由可生产（返回）的 MIME 类型 |
| Content-Type | Request Header | 客户端发送的请求体 MIME 类型  |
| Accept       | Request Header | 客户端希望接收的响应 MIME 类型 |

关键点：
- `Consumes` 用于验证请求的 `Content-Type` 是否匹配
- `Produces` 用于根据请求的 `Accept` 头选择响应合适的 `Content-Type`
- 响应的 `Content-Type` 在写入响应体时动态设置

### HTTP Server Plugins
