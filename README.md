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

### HTTP Server Plugins
