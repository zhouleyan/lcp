package o11y

import (
	"context"
	"fmt"
	"strconv"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// endpointStorage 监控端点资源的 REST 存储实现。
type endpointStorage struct {
	endpointStore EndpointStore
}

// NewEndpointStorage 创建监控端点 REST 存储。
func NewEndpointStorage(store EndpointStore) rest.StandardStorage {
	return &endpointStorage{endpointStore: store}
}

func (s *endpointStorage) NewObject() runtime.Object { return &Endpoint{} }

// List 获取监控端点列表。
// +openapi:summary=获取监控端点列表
func (s *endpointStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.endpointStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Endpoint, len(result.Items))
	for i, item := range result.Items {
		items[i] = endpointToAPI(&item)
	}

	return &EndpointList{
		TypeMeta:   runtime.TypeMeta{Kind: "EndpointList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Get 获取监控端点详情。
// +openapi:summary=获取监控端点详情
func (s *endpointStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["endpointId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid endpoint ID: %s", id), nil)
	}

	ep, err := s.endpointStore.GetByID(ctx, eid)
	if err != nil {
		return nil, err
	}

	result := endpointToAPI(ep)
	return &result, nil
}

// Create 创建监控端点。
// +openapi:summary=创建监控端点
func (s *endpointStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	ep, ok := obj.(*Endpoint)
	if !ok {
		return nil, fmt.Errorf("expected *Endpoint, got %T", obj)
	}

	if errs := ValidateEndpointCreate(ep.ObjectMeta.Name, &ep.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ep, nil
	}

	status := ep.Spec.Status
	if status == "" {
		status = "active"
	}
	public := true
	if ep.Spec.IsPublic != nil {
		public = *ep.Spec.IsPublic
	}

	created, err := s.endpointStore.Create(ctx, &DBEndpoint{
		Name:        ep.ObjectMeta.Name,
		Description: ep.Spec.Description,
		Public:      public,
		MetricsUrl:  ep.Spec.MetricsURL,
		LogsUrl:     ep.Spec.LogsURL,
		TracesUrl:   ep.Spec.TracesURL,
		ApmUrl:      ep.Spec.ApmURL,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	result := endpointToAPI(created)
	return &result, nil
}

// Update 全量更新监控端点。
// +openapi:summary=更新监控端点（全量）
func (s *endpointStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	ep, ok := obj.(*Endpoint)
	if !ok {
		return nil, fmt.Errorf("expected *Endpoint, got %T", obj)
	}

	if errs := ValidateEndpointUpdate(ep.ObjectMeta.Name, &ep.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ep, nil
	}

	id := options.PathParams["endpointId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid endpoint ID: %s", id), nil)
	}

	public := true
	if ep.Spec.IsPublic != nil {
		public = *ep.Spec.IsPublic
	}

	updated, err := s.endpointStore.Update(ctx, &DBEndpoint{
		ID:          eid,
		Name:        ep.ObjectMeta.Name,
		Description: ep.Spec.Description,
		Public:      public,
		MetricsUrl:  ep.Spec.MetricsURL,
		LogsUrl:     ep.Spec.LogsURL,
		TracesUrl:   ep.Spec.TracesURL,
		ApmUrl:      ep.Spec.ApmURL,
		Status:      ep.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	result := endpointToAPI(updated)
	return &result, nil
}

// Patch 部分更新监控端点。
// +openapi:summary=更新监控端点（部分）
func (s *endpointStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	ep, ok := obj.(*Endpoint)
	if !ok {
		return nil, fmt.Errorf("expected *Endpoint, got %T", obj)
	}

	id := options.PathParams["endpointId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid endpoint ID: %s", id), nil)
	}

	fields := endpointSpecToPatchFields(ep)

	patched, err := s.endpointStore.Patch(ctx, eid, fields)
	if err != nil {
		return nil, err
	}

	result := endpointToAPI(patched)
	return &result, nil
}

// Delete 删除单个监控端点。
// +openapi:summary=删除监控端点
func (s *endpointStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["endpointId"]
	eid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid endpoint ID: %s", id), nil)
	}

	return s.endpointStore.Delete(ctx, eid)
}

// DeleteCollection 批量删除监控端点。
// +openapi:summary=批量删除监控端点
func (s *endpointStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		eid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid endpoint ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, eid)
	}

	count, err := s.endpointStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// --- Helper functions ---

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

func endpointToAPI(ep *DBEndpoint) Endpoint {
	return Endpoint{
		TypeMeta: runtime.TypeMeta{Kind: "Endpoint"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(ep.ID, 10),
			Name:      ep.Name,
			CreatedAt: &ep.CreatedAt,
			UpdatedAt: &ep.UpdatedAt,
		},
		Spec: EndpointSpec{
			Description: ep.Description,
			IsPublic:    &ep.Public,
			MetricsURL:  ep.MetricsUrl,
			LogsURL:     ep.LogsUrl,
			TracesURL:   ep.TracesUrl,
			ApmURL:      ep.ApmUrl,
			Status:      ep.Status,
		},
	}
}

func endpointSpecToPatchFields(ep *Endpoint) map[string]any {
	fields := make(map[string]any)
	if ep.ObjectMeta.Name != "" {
		fields["name"] = ep.ObjectMeta.Name
	}
	if ep.Spec.IsPublic != nil {
		fields["public"] = *ep.Spec.IsPublic
	}
	// Always include description and optional URLs so they can be cleared to empty.
	fields["description"] = ep.Spec.Description
	if ep.Spec.MetricsURL != "" {
		fields["metricsUrl"] = ep.Spec.MetricsURL
	}
	fields["logsUrl"] = ep.Spec.LogsURL
	fields["tracesUrl"] = ep.Spec.TracesURL
	fields["apmUrl"] = ep.Spec.ApmURL
	if ep.Spec.Status != "" {
		fields["status"] = ep.Spec.Status
	}
	return fields
}

var parseID = rest.ParseID
