package rest

import (
	"context"

	"lcp.io/lcp/lib/runtime"
)

// ValidateObjectFunc 验证函数类型
type ValidateObjectFunc func(ctx context.Context, obj runtime.Object) error

// Getter 处理 GET 单个资源
type Getter interface {
	Get(ctx context.Context, id string) (runtime.Object, error)
}

// Lister 处理 GET 集合（支持过滤/分页/排序）
type Lister interface {
	List(ctx context.Context, options *ListOptions) (runtime.Object, error)
}

// Creator 处理 POST 创建
type Creator interface {
	Create(ctx context.Context, obj runtime.Object, validate ValidateObjectFunc, options *CreateOptions) (runtime.Object, error)
}

// Updater 处理 PUT 完整替换
type Updater interface {
	Update(ctx context.Context, id string, obj runtime.Object, validate ValidateObjectFunc, options *UpdateOptions) (runtime.Object, error)
}

// Patcher 处理 PATCH 部分更新（合并非空字段）
type Patcher interface {
	Patch(ctx context.Context, id string, obj runtime.Object, validate ValidateObjectFunc, options *PatchOptions) (runtime.Object, error)
}

// Deleter 处理 DELETE 单个资源
type Deleter interface {
	Delete(ctx context.Context, id string, validate ValidateObjectFunc, options *DeleteOptions) error
}

// CollectionDeleter 处理批量删除（通过显式 ID 列表）
type CollectionDeleter interface {
	DeleteCollection(ctx context.Context, ids []string, validate ValidateObjectFunc, options *DeleteOptions) (*DeletionResult, error)
}

// StandardStorage 组合所有操作
type StandardStorage interface {
	Getter
	Lister
	Creator
	Updater
	Patcher
	Deleter
	CollectionDeleter
}

// DeletionResult 批量删除结果
type DeletionResult struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []string `json:"failedIds,omitempty"`
}
