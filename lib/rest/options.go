package rest

import (
	"errors"
	"net/url"
	"strconv"
)

var (
	// ErrInvalidPage 页码无效
	ErrInvalidPage = errors.New("page must be greater than 0")
	// ErrInvalidPageSize 每页大小无效
	ErrInvalidPageSize = errors.New("page size must be between 1 and 100")
)

// SortOrder 排序方向
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// CreateOptions 创建选项
type CreateOptions struct {
	DryRun bool // 是否只验证不执行
}

// UpdateOptions 更新选项
type UpdateOptions struct {
	DryRun bool
}

// PatchOptions 补丁选项
type PatchOptions struct {
	DryRun bool
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	DryRun bool
}

// ListOptions 列表查询选项
type ListOptions struct {
	Filters    map[string]string // 过滤条件
	Pagination Pagination        // 分页参数
	SortBy     string            // 排序字段
	SortOrder  SortOrder         // 排序方向 (asc/desc)
}

// Pagination 分页参数
type Pagination struct {
	Page     int // 页码，从 1 开始
	PageSize int // 每页大小
}

// Validate 验证分页参数
func (p Pagination) Validate() error {
	if p.Page < 1 {
		return ErrInvalidPage
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		return ErrInvalidPageSize
	}
	return nil
}

// ReservedQueryParams 保留的查询参数名称，不应作为过滤条件
var ReservedQueryParams = map[string]bool{
	"page":      true,
	"pageSize":  true,
	"sortBy":    true,
	"sortOrder": true,
}

// ParseListOptions 从 URL 查询参数解析 ListOptions
func ParseListOptions(query url.Values) *ListOptions {
	options := &ListOptions{
		Filters: make(map[string]string),
		Pagination: Pagination{
			Page:     1,
			PageSize: 20,
		},
	}

	// 解析过滤条件（排除保留参数）
	for key, values := range query {
		if len(values) > 0 && !ReservedQueryParams[key] {
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
	if sortOrder := query.Get("sortOrder"); sortOrder != "" {
		options.SortOrder = SortOrder(sortOrder)
	}

	return options
}
