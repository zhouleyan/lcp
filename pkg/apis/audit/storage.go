package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// AuditLogStore abstracts database queries for audit logs.
type AuditLogStore interface {
	GetByID(ctx context.Context, id int64) (*DBAuditLog, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBAuditLog], error)
}

// +openapi:path=/logs
// +openapi:resource=AuditLog
type auditLogStorage struct {
	store AuditLogStore
}

// NewAuditLogStorage creates the REST storage for audit logs (read-only).
func NewAuditLogStorage(store AuditLogStore) rest.Storage {
	return &auditLogStorage{store: store}
}

// +openapi:summary=获取审计日志详情
func (s *auditLogStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["logId"]
	logID, err := rest.ParseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid log ID: %s", id), nil)
	}

	row, err := s.store.GetByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	return dbAuditLogToAPI(row), nil
}

// +openapi:summary=获取审计日志列表
func (s *auditLogStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.store.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]AuditLog, len(result.Items))
	for i := range result.Items {
		items[i] = *dbAuditLogToAPI(&result.Items[i])
	}

	return &AuditLogList{
		TypeMeta:   runtime.TypeMeta{Kind: "AuditLogList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

func dbAuditLogToAPI(row *DBAuditLog) *AuditLog {
	spec := AuditLogSpec{
		ID:           strconv.FormatInt(row.ID, 10),
		Username:     row.Username,
		EventType:    row.EventType,
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		Module:       row.Module,
		Scope:        row.Scope,
		HTTPMethod:   row.HttpMethod,
		HTTPPath:     row.HttpPath,
		StatusCode:   int(row.StatusCode),
		ClientIP:     row.ClientIp,
		UserAgent:    row.UserAgent,
		DurationMs:   int(row.DurationMs),
		Success:      row.Success,
		Detail:         nonNullJSON(row.Detail),
		ResponseDetail: nonNullJSON(row.ResponseDetail),
		CreatedAt:    row.CreatedAt.Format(time.RFC3339),
	}
	if row.UserID != nil {
		s := strconv.FormatInt(*row.UserID, 10)
		spec.UserID = &s
	}
	if row.WorkspaceID != nil {
		s := strconv.FormatInt(*row.WorkspaceID, 10)
		spec.WorkspaceID = &s
	}
	if row.NamespaceID != nil {
		s := strconv.FormatInt(*row.NamespaceID, 10)
		spec.NamespaceID = &s
	}
	return &AuditLog{
		TypeMeta: runtime.TypeMeta{Kind: "AuditLog"},
		Spec:     spec,
	}
}

func restOptionsToListQuery(options *rest.ListOptions) db.ListQuery {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}
	return query
}

// nonNullJSON returns nil for SQL NULL / JSON null values so omitempty works correctly.
func nonNullJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	return raw
}
