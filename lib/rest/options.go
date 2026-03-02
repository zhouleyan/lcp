package rest

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
	SortOrder  string            // 排序方向 (asc/desc)
}

// Pagination 分页参数
type Pagination struct {
	Page     int // 页码，从 1 开始
	PageSize int // 每页大小
}
