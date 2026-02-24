# Response Write

## 关键代码路径

### Handler 层
`staging/src/k8s.io/apiserver/pkg/endpoints/handlers/get.go`
- 调用 storage.Get() 获取对象
- 调用 transformResponseObject() 转换响应

### 转换层
`staging/src/k8s.io/apiserver/pkg/endpoints/handlers/response.go:314`

```shell
func transformResponseObject
```

### 写入层
`staging/src/k8s.io/apiserver/pkg/endpoints/handlers/responsewriters/writers.go:323`

```shell
func WriteObjectNegotiated
```

### 序列化层
`staging/src/k8s.io/apiserver/pkg/endpoints/handlers/responsewriters/writers.go:92`

```shell
func SerializeObject
```

### Content-Type 写入
`staging/src/k8s.io/apiserver/pkg/endpoints/handlers/responsewriters/writers.go:280`

```shell
func (w *deferredResponseWriter) unbufferedWrite
```