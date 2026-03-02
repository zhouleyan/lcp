## 任务 5: 实现通用 Handler 函数 - CreateResource 和 GetResource

**文件:**
- 修改: `lib/rest/handler.go`

**步骤 1: 添加 CreateResource 函数**

在 `lib/rest/handler.go` 添加：

```go
// CreateResource 返回处理资源创建的 http.HandlerFunc
func CreateResource(scope *RequestScope, storage Creater, validate ValidateObjectFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// 读取请求体
		body, err := io.ReadAll(req.Body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		defer req.Body.Close()

		// 反序列化对象
		obj, err := scope.Serializer.Decode(body, req.Header.Get("Content-Type"))
		if err != nil {
			scope.err(err, w, req)
			return
		}

		// 调用 storage 创建
		result, err := storage.Create(ctx, obj, validate, &CreateOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusCreated, result)
	}
}
```

**步骤 2: 添加 GetResource 函数**

```go
// GetResource 返回处理获取单个资源的 http.HandlerFunc
func GetResource(scope *RequestScope, storage Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params["userId"] // TODO: 通用化参数名
		if id == "" {
			scope.err(fmt.Errorf("missing resource id"), w, req)
			return
		}

		result, err := storage.Get(ctx, id)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}
```

**步骤 3: 提交**

```bash
git add lib/rest/handler.go
git commit -m "feat(rest): add CreateResource and GetResource handlers

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 任务 6: 实现通用 Handler 函数 - ListResource 和 UpdateResource

**文件:**
- 修改: `lib/rest/handler.go`

**步骤 1: 添加 ListResource 函数**

```go
// ListResource 返回处理资源列表查询的 http.HandlerFunc
func ListResource(scope *RequestScope, storage Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// 解析查询参数
		query := req.URL.Query()
		options := &ListOptions{
			Filters: make(map[string]string),
			Pagination: Pagination{
				Page:     1,
				PageSize: 20,
			},
		}

		// 解析过滤条件
		for key, values := range query {
			if len(values) > 0 && key != "page" && key != "pageSize" && key != "sortBy" && key != "sortOrder" {
				options.Filters[key] = values[0]
			}
		}

		// 解析分页
		if page := query.Get("page"); page != "" {
			if p, err := strconv.Atoi(page); err == nil && p > 0 {
				options.Pagination.Page = p
			}
		}
		if pageSize := query.Get("pageSize"); pageSize != "" {
			if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
				options.Pagination.PageSize = ps
			}
		}

		// 解析排序
		options.SortBy = query.Get("sortBy")
		options.SortOrder = query.Get("sortOrder")

		result, err := storage.List(ctx, options)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}
```

**步骤 2: 添加 UpdateResource 函数**

```go
// UpdateResource 返回处理资源完整更新的 http.HandlerFunc
func UpdateResource(scope *RequestScope, storage Updater, validate ValidateObjectFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params["userId"] // TODO: 通用化参数名
		if id == "" {
			scope.err(fmt.Errorf("missing resource id"), w, req)
			return
		}

		// 读取请求体
		body, err := io.ReadAll(req.Body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		defer req.Body.Close()

		// 反序列化对象
		obj, err := scope.Serializer.Decode(body, req.Header.Get("Content-Type"))
		if err != nil {
			scope.err(err, w, req)
			return
		}

		// 调用 storage 更新
		result, err := storage.Update(ctx, id, obj, validate, &UpdateOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}
```

**步骤 3: 添加必要的 import**

确保有 `strconv` 和 `fmt` 的 import

**步骤 4: 提交**

```bash
git add lib/rest/handler.go
git commit -m "feat(rest): add ListResource and UpdateResource handlers

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 任务 7: 实现通用 Handler 函数 - PatchResource, DeleteResource, DeleteCollection

**文件:**
- 修改: `lib/rest/handler.go`

**步骤 1: 添加 PatchResource 函数**

```go
// PatchResource 返回处理资源部分更新的 http.HandlerFunc
func PatchResource(scope *RequestScope, storage Patcher, validate ValidateObjectFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params["userId"] // TODO: 通用化参数名
		if id == "" {
			scope.err(fmt.Errorf("missing resource id"), w, req)
			return
		}

		// 读取请求体
		body, err := io.ReadAll(req.Body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		defer req.Body.Close()

		// 反序列化对象
		obj, err := scope.Serializer.Decode(body, req.Header.Get("Content-Type"))
		if err != nil {
			scope.err(err, w, req)
			return
		}

		// 调用 storage patch
		result, err := storage.Patch(ctx, id, obj, validate, &PatchOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}
```

**步骤 2: 添加 DeleteResource 函数**

```go
// DeleteResource 返回处理删除单个资源的 http.HandlerFunc
func DeleteResource(scope *RequestScope, storage Deleter, validate ValidateObjectFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params["userId"] // TODO: 通用化参数名
		if id == "" {
			scope.err(fmt.Errorf("missing resource id"), w, req)
			return
		}

		err := storage.Delete(ctx, id, validate, &DeleteOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
```

**步骤 3: 添加 DeleteCollection 函数**

```go
// DeleteCollectionRequest 批量删除请求
type DeleteCollectionRequest struct {
	IDs []string `json:"ids"`
}

// DeleteCollection 返回处理批量删除的 http.HandlerFunc
func DeleteCollection(scope *RequestScope, storage CollectionDeleter, validate ValidateObjectFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// 读取请求体
		body, err := io.ReadAll(req.Body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		defer req.Body.Close()

		// 解析 ID 列表
		var deleteReq DeleteCollectionRequest
		if err := json.Unmarshal(body, &deleteReq); err != nil {
			scope.err(err, w, req)
			return
		}

		if len(deleteReq.IDs) == 0 {
			scope.err(fmt.Errorf("no ids provided"), w, req)
			return
		}

		// 调用 storage 批量删除
		result, err := storage.DeleteCollection(ctx, deleteReq.IDs, validate, &DeleteOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}
```

**步骤 4: 添加 json import**

**步骤 5: 提交**

```bash
git add lib/rest/handler.go
git commit -m "feat(rest): add PatchResource, DeleteResource, DeleteCollection handlers

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---
