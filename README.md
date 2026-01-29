# lcp
LCP

## Start LCP Server

```bash
lcp-server \
--httpListenerAddr=:8421 \
--httpListenerAddr=:8422 \
--httpListenerAddr=:8423
```

## TODO

### REST API Routes

http.Server.Handler -> APIServerHandler -> goRestfulContainer := restful.NewContainer(...)

APIServerHandler.ServeHTTP(...) -> FullHandlerChain.ServeHTTP(...) -> director.ServeHTTP(...) -> goRestfulContainer.Dispatch(w, req) -> c.dispatch(w, req)

### HTTP Server Plugins
