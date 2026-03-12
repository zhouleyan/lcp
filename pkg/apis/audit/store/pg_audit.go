package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	apierrors "lcp.io/lcp/lib/api/errors"
	libaudit "lcp.io/lcp/lib/audit"
	"lcp.io/lcp/pkg/apis/audit"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgAuditLogStore struct {
	queries *generated.Queries
}

// NewPGAuditLogStore creates a PostgreSQL-backed store that implements
// both audit.AuditLogStore (query) and libaudit.Sink (batch write).
func NewPGAuditLogStore(queries *generated.Queries) *pgAuditLogStore {
	return &pgAuditLogStore{queries: queries}
}

// --- libaudit.Sink implementation ---

func (s *pgAuditLogStore) BatchCreate(ctx context.Context, events []libaudit.Event) error {
	for _, e := range events {
		detail := e.Detail
		if detail == nil {
			detail = json.RawMessage("null")
		}
		responseDetail := e.ResponseDetail
		if responseDetail == nil {
			responseDetail = json.RawMessage("null")
		}
		err := s.queries.CreateAuditLog(ctx, generated.CreateAuditLogParams{
			UserID:         e.UserID,
			Username:       e.Username,
			EventType:      e.EventType,
			Action:         e.Action,
			ResourceType:   e.ResourceType,
			ResourceID:     e.ResourceID,
			Module:         e.Module,
			Scope:          e.Scope,
			WorkspaceID:    e.WorkspaceID,
			NamespaceID:    e.NamespaceID,
			HttpMethod:     e.HTTPMethod,
			HttpPath:       e.HTTPPath,
			StatusCode:     int32(e.StatusCode),
			ClientIp:       e.ClientIP,
			UserAgent:      e.UserAgent,
			DurationMs:     int32(e.DurationMs),
			Success:        e.Success,
			Detail:         detail,
			ResponseDetail: responseDetail,
			CreatedAt:      e.CreatedAt,
		})
		if err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}
	}
	return nil
}

// --- audit.AuditLogStore implementation ---

func (s *pgAuditLogStore) GetByID(ctx context.Context, id int64) (*audit.DBAuditLog, error) {
	row, err := s.queries.GetAuditLog(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("audit log", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	return &row, nil
}

func (s *pgAuditLogStore) List(ctx context.Context, query db.ListQuery) (*db.ListResult[audit.DBAuditLog], error) {
	offset, limit := db.PaginationToOffsetLimit(query.Pagination)

	filterParams := buildFilterParams(query.Filters)

	count, err := s.queries.CountAuditLogs(ctx, generated.CountAuditLogsParams{
		UserID:       filterParams.UserID,
		EventType:    filterParams.EventType,
		Action:       filterParams.Action,
		ResourceType: filterParams.ResourceType,
		ResourceID:   filterParams.ResourceID,
		Module:       filterParams.Module,
		WorkspaceID:  filterParams.WorkspaceID,
		NamespaceID:  filterParams.NamespaceID,
		Success:      filterParams.Success,
		StatusCode:   filterParams.StatusCode,
		StartTime:    filterParams.StartTime,
		EndTime:      filterParams.EndTime,
		Search:       filterParams.Search,
	})
	if err != nil {
		return nil, fmt.Errorf("count audit logs: %w", err)
	}

	sortField := query.SortBy
	if sortField == "" {
		sortField = "created_at"
	}
	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListAuditLogs(ctx, generated.ListAuditLogsParams{
		UserID:       filterParams.UserID,
		EventType:    filterParams.EventType,
		Action:       filterParams.Action,
		ResourceType: filterParams.ResourceType,
		ResourceID:   filterParams.ResourceID,
		Module:       filterParams.Module,
		WorkspaceID:  filterParams.WorkspaceID,
		NamespaceID:  filterParams.NamespaceID,
		Success:      filterParams.Success,
		StatusCode:   filterParams.StatusCode,
		StartTime:    filterParams.StartTime,
		EndTime:      filterParams.EndTime,
		Search:       filterParams.Search,
		SortField:    sortField,
		SortOrder:    sortOrder,
		PageOffset:   offset,
		PageSize:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}

	return &db.ListResult[audit.DBAuditLog]{
		Items:      rows,
		TotalCount: count,
	}, nil
}

func buildFilterParams(filters map[string]any) generated.ListAuditLogsParams {
	return generated.ListAuditLogsParams{
		UserID:       filterInt64(filters, "userId"),
		EventType:    filterStr(filters, "eventType"),
		Action:       filterStr(filters, "action"),
		ResourceType: filterStr(filters, "resourceType"),
		ResourceID:   filterStr(filters, "resourceId"),
		Module:       filterStr(filters, "module"),
		WorkspaceID:  filterInt64(filters, "workspaceId"),
		NamespaceID:  filterInt64(filters, "namespaceId"),
		Success:      filterBool(filters, "success"),
		StatusCode:   filterInt32(filters, "statusCode"),
		StartTime:    filterTime(filters, "startTime"),
		EndTime:      filterTime(filters, "endTime"),
		Search:       filterStr(filters, "search"),
	}
}
