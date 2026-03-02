package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"lcp.io/lcp/lib/runtime"
)

// RequestScope encapsulates common fields across all RESTful handler methods
type RequestScope struct {
	Serializer runtime.NegotiatedSerializer
}

func (scope *RequestScope) err(err error, w http.ResponseWriter, r *http.Request) {
	ErrorNegotiated(w, r, scope.Serializer, err)
}

// pathParamsFromRequest extracts the path parameters map from the request context.
// Returns an empty map if no path parameters are set.
func pathParamsFromRequest(req *http.Request) map[string]string {
	v := req.Context().Value(PathParamsKey)
	if v == nil {
		return map[string]string{}
	}
	params, ok := v.(map[string]string)
	if !ok {
		return map[string]string{}
	}
	return params
}

// HandlerFunc is the unified function signature for all request handlers.
// Params are extracted from path; body is the raw request body (nil for bodiless requests).
type HandlerFunc func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error)

// Handle returns an http.HandlerFunc that:
//  1. Extracts path params from context
//  2. Reads request body (if present)
//  3. Calls fn
//  4. Writes the response with the given statusCode (or 204 if result is nil)
func Handle(scope *RequestScope, statusCode int, fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		var body []byte
		if req.Body != nil && req.ContentLength != 0 {
			var err error
			body, err = io.ReadAll(req.Body)
			if err != nil {
				scope.err(err, w, req)
				return
			}
			defer req.Body.Close()
		}

		result, err := fn(ctx, params, body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		if result == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		transformResponseObject(scope, req, w, statusCode, result)
	}
}

func transformResponseObject(
	scope *RequestScope,
	req *http.Request,
	w http.ResponseWriter,
	statusCode int,
	result runtime.Object,
) {
	WriteObjectNegotiated(scope.Serializer, w, req, statusCode, result)
}

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
