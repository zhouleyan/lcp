package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// StatusResponse is a generic operation result for infra actions.
type StatusResponse struct {
	runtime.TypeMeta `json:",inline"`
	Status           string `json:"status"`
	Message          string `json:"message"`
}

func (s *StatusResponse) GetTypeMeta() *runtime.TypeMeta { return &s.TypeMeta }

// ===== hostStorage 平台级主机存储 =====

// hostStorage 平台级主机资源的 REST 存储实现，支持 CRUD 和批量删除。
type hostStorage struct {
	hostStore HostStore
}

// NewHostStorage 创建平台级主机 REST 存储。
func NewHostStorage(hostStore HostStore) rest.StandardStorage {
	return &hostStorage{hostStore: hostStore}
}

func (s *hostStorage) NewObject() runtime.Object { return &Host{} }

// Get 获取主机详情。
// +openapi:summary=获取主机详情
func (s *hostStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	host, err := s.hostStore.GetByID(ctx, hid)
	if err != nil {
		return nil, err
	}

	return hostWithEnvToAPI(host), nil
}

// List 获取平台级主机列表。
// +openapi:summary=获取主机列表
func (s *hostStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.hostStore.ListPlatform(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Host, len(result.Items))
	for i, item := range result.Items {
		items[i] = hostPlatformRowToAPI(&item)
	}

	return &HostList{
		TypeMeta:   runtime.TypeMeta{Kind: "HostList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建平台级主机。
// +openapi:summary=创建主机
func (s *hostStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	if errs := ValidateHostCreate(host.ObjectMeta.Name, &host.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return host, nil
	}

	status := host.Spec.Status
	if status == "" {
		status = "active"
	}

	labels := json.RawMessage("{}")
	if host.Spec.Labels != nil {
		raw, err := json.Marshal(host.Spec.Labels)
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid labels", nil)
		}
		labels = raw
	}

	created, err := s.hostStore.Create(ctx, &DBHost{
		Name:     host.ObjectMeta.Name,
		Hostname: host.Spec.Hostname,
		IpAddress: host.Spec.IPAddress,
		Os:       host.Spec.OS,
		Arch:     host.Spec.Arch,
		CpuCores: host.Spec.CPUCores,
		MemoryMb: host.Spec.MemoryMB,
		DiskGb:   host.Spec.DiskGB,
		Labels:   labels,
		Scope:    ScopePlatform,
		Status:   status,
	})
	if err != nil {
		return nil, err
	}

	return hostToAPI(created), nil
}

// Update 全量更新主机信息。
// +openapi:summary=更新主机信息（全量）
func (s *hostStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	if errs := ValidateHostUpdate(&host.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return host, nil
	}

	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	labels := json.RawMessage("{}")
	if host.Spec.Labels != nil {
		raw, err := json.Marshal(host.Spec.Labels)
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid labels", nil)
		}
		labels = raw
	}

	updated, err := s.hostStore.Update(ctx, &DBHost{
		ID:        hid,
		Name:      host.ObjectMeta.Name,
		Hostname:  host.Spec.Hostname,
		IpAddress: host.Spec.IPAddress,
		Os:        host.Spec.OS,
		Arch:      host.Spec.Arch,
		CpuCores:  host.Spec.CPUCores,
		MemoryMb:  host.Spec.MemoryMB,
		DiskGb:    host.Spec.DiskGB,
		Labels:    labels,
		Status:    host.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return hostToAPI(updated), nil
}

// Patch 部分更新主机信息。
// +openapi:summary=更新主机信息（部分）
func (s *hostStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	id := options.PathParams["hostId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	fields := hostSpecToPatchFields(host)

	patched, err := s.hostStore.Patch(ctx, hid, fields)
	if err != nil {
		return nil, err
	}

	return hostToAPI(patched), nil
}

// Delete 删除单个主机。
// +openapi:summary=删除主机
func (s *hostStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	return s.hostStore.Delete(ctx, hid)
}

// DeleteCollection 批量删除主机。
// +openapi:summary=批量删除主机
func (s *hostStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		hid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, hid)
	}

	count, err := s.hostStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== workspaceHostStorage 租户级主机存储 =====

// workspaceHostStorage 租户级主机资源的 REST 存储实现。
// +openapi:path=/workspaces/{workspaceId}/hosts
type workspaceHostStorage struct {
	hostStore HostStore
}

// NewWorkspaceHostStorage 创建租户级主机 REST 存储。
func NewWorkspaceHostStorage(hostStore HostStore) rest.StandardStorage {
	return &workspaceHostStorage{hostStore: hostStore}
}

func (s *workspaceHostStorage) NewObject() runtime.Object { return &Host{} }

// Get 获取租户下的主机详情。
// +openapi:summary=获取主机详情
// +openapi:summary.workspaces.hosts=获取租户下的主机详情
func (s *workspaceHostStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	host, err := s.hostStore.GetByID(ctx, hid)
	if err != nil {
		return nil, err
	}

	return hostWithEnvToAPI(host), nil
}

// List 获取租户下的主机列表（含自有和被分配的主机）。
// +openapi:summary=获取主机列表
// +openapi:summary.workspaces.hosts=获取租户下的主机列表
func (s *workspaceHostStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.hostStore.ListByWorkspaceID(ctx, wsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]Host, len(result.Items))
	for i, item := range result.Items {
		items[i] = hostWorkspaceRowToAPI(&item)
	}

	return &HostList{
		TypeMeta:   runtime.TypeMeta{Kind: "HostList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 在租户下创建主机。
// +openapi:summary=创建主机
// +openapi:summary.workspaces.hosts=在租户下创建主机
func (s *workspaceHostStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	if errs := ValidateHostCreate(host.ObjectMeta.Name, &host.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return host, nil
	}

	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	status := host.Spec.Status
	if status == "" {
		status = "active"
	}

	labels := json.RawMessage("{}")
	if host.Spec.Labels != nil {
		raw, err := json.Marshal(host.Spec.Labels)
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid labels", nil)
		}
		labels = raw
	}

	created, err := s.hostStore.Create(ctx, &DBHost{
		Name:        host.ObjectMeta.Name,
		Hostname:    host.Spec.Hostname,
		IpAddress:   host.Spec.IPAddress,
		Os:          host.Spec.OS,
		Arch:        host.Spec.Arch,
		CpuCores:    host.Spec.CPUCores,
		MemoryMb:    host.Spec.MemoryMB,
		DiskGb:      host.Spec.DiskGB,
		Labels:      labels,
		Scope:       ScopeWorkspace,
		WorkspaceID: &wsID,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return hostToAPI(created), nil
}

// Update 全量更新租户下的主机信息。
// +openapi:summary=更新主机信息（全量）
// +openapi:summary.workspaces.hosts=更新租户下的主机信息（全量）
func (s *workspaceHostStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	if errs := ValidateHostUpdate(&host.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return host, nil
	}

	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	labels := json.RawMessage("{}")
	if host.Spec.Labels != nil {
		raw, err := json.Marshal(host.Spec.Labels)
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid labels", nil)
		}
		labels = raw
	}

	updated, err := s.hostStore.Update(ctx, &DBHost{
		ID:        hid,
		Name:      host.ObjectMeta.Name,
		Hostname:  host.Spec.Hostname,
		IpAddress: host.Spec.IPAddress,
		Os:        host.Spec.OS,
		Arch:      host.Spec.Arch,
		CpuCores:  host.Spec.CPUCores,
		MemoryMb:  host.Spec.MemoryMB,
		DiskGb:    host.Spec.DiskGB,
		Labels:    labels,
		Status:    host.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return hostToAPI(updated), nil
}

// Patch 部分更新租户下的主机信息。
// +openapi:summary=更新主机信息（部分）
// +openapi:summary.workspaces.hosts=更新租户下的主机信息（部分）
func (s *workspaceHostStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	id := options.PathParams["hostId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	fields := hostSpecToPatchFields(host)

	patched, err := s.hostStore.Patch(ctx, hid, fields)
	if err != nil {
		return nil, err
	}

	return hostToAPI(patched), nil
}

// Delete 删除租户下的主机。
// +openapi:summary=删除主机
// +openapi:summary.workspaces.hosts=删除租户下的主机
func (s *workspaceHostStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	return s.hostStore.Delete(ctx, hid)
}

// DeleteCollection 批量删除租户下的主机。
// +openapi:summary=批量删除主机
// +openapi:summary.workspaces.hosts=批量删除租户下的主机
func (s *workspaceHostStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		hid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, hid)
	}

	count, err := s.hostStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== namespaceHostStorage 项目级主机存储 =====

// namespaceHostStorage 项目级主机资源的 REST 存储实现。
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/hosts
type namespaceHostStorage struct {
	hostStore HostStore
}

// NewNamespaceHostStorage 创建项目级主机 REST 存储。
func NewNamespaceHostStorage(hostStore HostStore) rest.StandardStorage {
	return &namespaceHostStorage{hostStore: hostStore}
}

func (s *namespaceHostStorage) NewObject() runtime.Object { return &Host{} }

// Get 获取项目下的主机详情。
// +openapi:summary=获取主机详情
// +openapi:summary.workspaces.namespaces.hosts=获取项目下的主机详情
func (s *namespaceHostStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	host, err := s.hostStore.GetByID(ctx, hid)
	if err != nil {
		return nil, err
	}

	return hostWithEnvToAPI(host), nil
}

// List 获取项目下的主机列表（含自有和被分配的主机）。
// +openapi:summary=获取主机列表
// +openapi:summary.workspaces.namespaces.hosts=获取项目下的主机列表
func (s *namespaceHostStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.hostStore.ListByNamespaceID(ctx, nsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]Host, len(result.Items))
	for i, item := range result.Items {
		items[i] = hostNamespaceRowToAPI(&item)
	}

	return &HostList{
		TypeMeta:   runtime.TypeMeta{Kind: "HostList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 在项目下创建主机。
// +openapi:summary=创建主机
// +openapi:summary.workspaces.namespaces.hosts=在项目下创建主机
func (s *namespaceHostStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	if errs := ValidateHostCreate(host.ObjectMeta.Name, &host.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return host, nil
	}

	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	status := host.Spec.Status
	if status == "" {
		status = "active"
	}

	labels := json.RawMessage("{}")
	if host.Spec.Labels != nil {
		raw, err := json.Marshal(host.Spec.Labels)
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid labels", nil)
		}
		labels = raw
	}

	created, err := s.hostStore.Create(ctx, &DBHost{
		Name:        host.ObjectMeta.Name,
		Hostname:    host.Spec.Hostname,
		IpAddress:   host.Spec.IPAddress,
		Os:          host.Spec.OS,
		Arch:        host.Spec.Arch,
		CpuCores:    host.Spec.CPUCores,
		MemoryMb:    host.Spec.MemoryMB,
		DiskGb:      host.Spec.DiskGB,
		Labels:      labels,
		Scope:       ScopeNamespace,
		NamespaceID: &nsID,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return hostToAPI(created), nil
}

// Update 全量更新项目下的主机信息。
// +openapi:summary=更新主机信息（全量）
// +openapi:summary.workspaces.namespaces.hosts=更新项目下的主机信息（全量）
func (s *namespaceHostStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	if errs := ValidateHostUpdate(&host.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return host, nil
	}

	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	labels := json.RawMessage("{}")
	if host.Spec.Labels != nil {
		raw, err := json.Marshal(host.Spec.Labels)
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid labels", nil)
		}
		labels = raw
	}

	updated, err := s.hostStore.Update(ctx, &DBHost{
		ID:        hid,
		Name:      host.ObjectMeta.Name,
		Hostname:  host.Spec.Hostname,
		IpAddress: host.Spec.IPAddress,
		Os:        host.Spec.OS,
		Arch:      host.Spec.Arch,
		CpuCores:  host.Spec.CPUCores,
		MemoryMb:  host.Spec.MemoryMB,
		DiskGb:    host.Spec.DiskGB,
		Labels:    labels,
		Status:    host.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return hostToAPI(updated), nil
}

// Patch 部分更新项目下的主机信息。
// +openapi:summary=更新主机信息（部分）
// +openapi:summary.workspaces.namespaces.hosts=更新项目下的主机信息（部分）
func (s *namespaceHostStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected *Host, got %T", obj)
	}

	id := options.PathParams["hostId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	hid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	fields := hostSpecToPatchFields(host)

	patched, err := s.hostStore.Patch(ctx, hid, fields)
	if err != nil {
		return nil, err
	}

	return hostToAPI(patched), nil
}

// Delete 删除项目下的主机。
// +openapi:summary=删除主机
// +openapi:summary.workspaces.namespaces.hosts=删除项目下的主机
func (s *namespaceHostStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["hostId"]
	hid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
	}

	return s.hostStore.Delete(ctx, hid)
}

// DeleteCollection 批量删除项目下的主机。
// +openapi:summary=批量删除主机
// +openapi:summary.workspaces.namespaces.hosts=批量删除项目下的主机
func (s *namespaceHostStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		hid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, hid)
	}

	count, err := s.hostStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== environmentStorage 平台级环境存储 =====

// environmentStorage 平台级环境资源的 REST 存储实现，支持 CRUD 和批量删除。
type environmentStorage struct {
	envStore EnvironmentStore
}

// NewEnvironmentStorage 创建平台级环境 REST 存储。
func NewEnvironmentStorage(envStore EnvironmentStore) rest.StandardStorage {
	return &environmentStorage{envStore: envStore}
}

func (s *environmentStorage) NewObject() runtime.Object { return &Environment{} }

// Get 获取环境详情。
// +openapi:summary=获取环境详情
func (s *environmentStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	env, err := s.envStore.GetByID(ctx, eid)
	if err != nil {
		return nil, err
	}

	return envWithCountsToAPI(env), nil
}

// List 获取平台级环境列表。
// +openapi:summary=获取环境列表
func (s *environmentStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.envStore.ListPlatform(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Environment, len(result.Items))
	for i, item := range result.Items {
		items[i] = envPlatformRowToAPI(&item)
	}

	return &EnvironmentList{
		TypeMeta:   runtime.TypeMeta{Kind: "EnvironmentList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建平台级环境。
// +openapi:summary=创建环境
func (s *environmentStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	if errs := ValidateEnvironmentCreate(env.ObjectMeta.Name, &env.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return env, nil
	}

	status := env.Spec.Status
	if status == "" {
		status = "active"
	}

	envType := env.Spec.EnvType
	if envType == "" {
		envType = "custom"
	}

	created, err := s.envStore.Create(ctx, &DBEnvironment{
		Name:        env.ObjectMeta.Name,
		DisplayName: env.Spec.DisplayName,
		Description: env.Spec.Description,
		EnvType:     envType,
		Scope:       ScopePlatform,
		WorkspaceID: nil,
		NamespaceID: nil,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return envToAPI(created), nil
}

// Update 全量更新环境信息。
// +openapi:summary=更新环境信息（全量）
func (s *environmentStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	if errs := ValidateEnvironmentUpdate(&env.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return env, nil
	}

	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	updated, err := s.envStore.Update(ctx, &DBEnvironment{
		ID:          eid,
		Name:        env.ObjectMeta.Name,
		DisplayName: env.Spec.DisplayName,
		Description: env.Spec.Description,
		EnvType:     env.Spec.EnvType,
		Status:      env.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return envToAPI(updated), nil
}

// Patch 部分更新环境信息。
// +openapi:summary=更新环境信息（部分）
func (s *environmentStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	id := options.PathParams["environmentId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	fields := make(map[string]any)
	if env.ObjectMeta.Name != "" {
		fields["name"] = env.ObjectMeta.Name
	}
	if env.Spec.DisplayName != "" {
		fields["display_name"] = env.Spec.DisplayName
	}
	if env.Spec.Description != "" {
		fields["description"] = env.Spec.Description
	}
	if env.Spec.EnvType != "" {
		fields["env_type"] = env.Spec.EnvType
	}
	if env.Spec.Status != "" {
		fields["status"] = env.Spec.Status
	}

	patched, err := s.envStore.Patch(ctx, eid, fields)
	if err != nil {
		return nil, err
	}

	return envToAPI(patched), nil
}

// Delete 删除单个环境。
// +openapi:summary=删除环境
func (s *environmentStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	return s.envStore.Delete(ctx, eid)
}

// DeleteCollection 批量删除环境。
// +openapi:summary=批量删除环境
func (s *environmentStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
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
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, eid)
	}

	count, err := s.envStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== workspaceEnvironmentStorage 租户级环境存储 =====

// workspaceEnvironmentStorage 租户级环境资源的 REST 存储实现。
// +openapi:path=/workspaces/{workspaceId}/environments
type workspaceEnvironmentStorage struct {
	envStore EnvironmentStore
}

// NewWorkspaceEnvironmentStorage 创建租户级环境 REST 存储。
func NewWorkspaceEnvironmentStorage(envStore EnvironmentStore) rest.StandardStorage {
	return &workspaceEnvironmentStorage{envStore: envStore}
}

func (s *workspaceEnvironmentStorage) NewObject() runtime.Object { return &Environment{} }

// Get 获取租户下的环境详情。
// +openapi:summary=获取环境详情
// +openapi:summary.workspaces.environments=获取租户下的环境详情
func (s *workspaceEnvironmentStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	env, err := s.envStore.GetByID(ctx, eid)
	if err != nil {
		return nil, err
	}

	return envWithCountsToAPI(env), nil
}

// List 获取租户下的环境列表。
// +openapi:summary=获取环境列表
// +openapi:summary.workspaces.environments=获取租户下的环境列表
func (s *workspaceEnvironmentStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.envStore.ListByWorkspaceID(ctx, wsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]Environment, len(result.Items))
	for i, item := range result.Items {
		items[i] = envWorkspaceRowToAPI(&item)
	}

	return &EnvironmentList{
		TypeMeta:   runtime.TypeMeta{Kind: "EnvironmentList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 在租户下创建环境。
// +openapi:summary=创建环境
// +openapi:summary.workspaces.environments=在租户下创建环境
func (s *workspaceEnvironmentStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	if errs := ValidateEnvironmentCreate(env.ObjectMeta.Name, &env.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return env, nil
	}

	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	status := env.Spec.Status
	if status == "" {
		status = "active"
	}

	envType := env.Spec.EnvType
	if envType == "" {
		envType = "custom"
	}

	created, err := s.envStore.Create(ctx, &DBEnvironment{
		Name:        env.ObjectMeta.Name,
		DisplayName: env.Spec.DisplayName,
		Description: env.Spec.Description,
		EnvType:     envType,
		Scope:       ScopeWorkspace,
		WorkspaceID: &wsID,
		NamespaceID: nil,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return envToAPI(created), nil
}

// Update 全量更新租户下的环境信息。
// +openapi:summary=更新环境信息（全量）
// +openapi:summary.workspaces.environments=更新租户下的环境信息（全量）
func (s *workspaceEnvironmentStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	if errs := ValidateEnvironmentUpdate(&env.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return env, nil
	}

	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	updated, err := s.envStore.Update(ctx, &DBEnvironment{
		ID:          eid,
		Name:        env.ObjectMeta.Name,
		DisplayName: env.Spec.DisplayName,
		Description: env.Spec.Description,
		EnvType:     env.Spec.EnvType,
		Status:      env.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return envToAPI(updated), nil
}

// Patch 部分更新租户下的环境信息。
// +openapi:summary=更新环境信息（部分）
// +openapi:summary.workspaces.environments=更新租户下的环境信息（部分）
func (s *workspaceEnvironmentStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	id := options.PathParams["environmentId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	fields := envSpecToPatchFields(env)

	patched, err := s.envStore.Patch(ctx, eid, fields)
	if err != nil {
		return nil, err
	}

	return envToAPI(patched), nil
}

// Delete 删除租户下的环境。
// +openapi:summary=删除环境
// +openapi:summary.workspaces.environments=删除租户下的环境
func (s *workspaceEnvironmentStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	return s.envStore.Delete(ctx, eid)
}

// DeleteCollection 批量删除租户下的环境。
// +openapi:summary=批量删除环境
// +openapi:summary.workspaces.environments=批量删除租户下的环境
func (s *workspaceEnvironmentStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
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
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, eid)
	}

	count, err := s.envStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== namespaceEnvironmentStorage 项目级环境存储 =====

// namespaceEnvironmentStorage 项目级环境资源的 REST 存储实现。
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/environments
type namespaceEnvironmentStorage struct {
	envStore EnvironmentStore
}

// NewNamespaceEnvironmentStorage 创建项目级环境 REST 存储。
func NewNamespaceEnvironmentStorage(envStore EnvironmentStore) rest.StandardStorage {
	return &namespaceEnvironmentStorage{envStore: envStore}
}

func (s *namespaceEnvironmentStorage) NewObject() runtime.Object { return &Environment{} }

// Get 获取项目下的环境详情。
// +openapi:summary=获取环境详情
// +openapi:summary.workspaces.namespaces.environments=获取项目下的环境详情
func (s *namespaceEnvironmentStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	env, err := s.envStore.GetByID(ctx, eid)
	if err != nil {
		return nil, err
	}

	return envWithCountsToAPI(env), nil
}

// List 获取项目下的环境列表。
// +openapi:summary=获取环境列表
// +openapi:summary.workspaces.namespaces.environments=获取项目下的环境列表
func (s *namespaceEnvironmentStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.envStore.ListByNamespaceID(ctx, nsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]Environment, len(result.Items))
	for i, item := range result.Items {
		items[i] = envNamespaceRowToAPI(&item)
	}

	return &EnvironmentList{
		TypeMeta:   runtime.TypeMeta{Kind: "EnvironmentList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 在项目下创建环境。
// +openapi:summary=创建环境
// +openapi:summary.workspaces.namespaces.environments=在项目下创建环境
func (s *namespaceEnvironmentStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	if errs := ValidateEnvironmentCreate(env.ObjectMeta.Name, &env.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return env, nil
	}

	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	status := env.Spec.Status
	if status == "" {
		status = "active"
	}

	envType := env.Spec.EnvType
	if envType == "" {
		envType = "custom"
	}

	created, err := s.envStore.Create(ctx, &DBEnvironment{
		Name:        env.ObjectMeta.Name,
		DisplayName: env.Spec.DisplayName,
		Description: env.Spec.Description,
		EnvType:     envType,
		Scope:       ScopeNamespace,
		WorkspaceID: nil,
		NamespaceID: &nsID,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return envToAPI(created), nil
}

// Update 全量更新项目下的环境信息。
// +openapi:summary=更新环境信息（全量）
// +openapi:summary.workspaces.namespaces.environments=更新项目下的环境信息（全量）
func (s *namespaceEnvironmentStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	if errs := ValidateEnvironmentUpdate(&env.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return env, nil
	}

	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	updated, err := s.envStore.Update(ctx, &DBEnvironment{
		ID:          eid,
		Name:        env.ObjectMeta.Name,
		DisplayName: env.Spec.DisplayName,
		Description: env.Spec.Description,
		EnvType:     env.Spec.EnvType,
		Status:      env.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return envToAPI(updated), nil
}

// Patch 部分更新项目下的环境信息。
// +openapi:summary=更新环境信息（部分）
// +openapi:summary.workspaces.namespaces.environments=更新项目下的环境信息（部分）
func (s *namespaceEnvironmentStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected *Environment, got %T", obj)
	}

	id := options.PathParams["environmentId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	eid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	fields := envSpecToPatchFields(env)

	patched, err := s.envStore.Patch(ctx, eid, fields)
	if err != nil {
		return nil, err
	}

	return envToAPI(patched), nil
}

// Delete 删除项目下的环境。
// +openapi:summary=删除环境
// +openapi:summary.workspaces.namespaces.environments=删除项目下的环境
func (s *namespaceEnvironmentStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["environmentId"]
	eid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
	}

	return s.envStore.Delete(ctx, eid)
}

// DeleteCollection 批量删除项目下的环境。
// +openapi:summary=批量删除环境
// +openapi:summary.workspaces.namespaces.environments=批量删除项目下的环境
func (s *namespaceEnvironmentStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
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
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, eid)
	}

	count, err := s.envStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== assign 主机分配操作 =====

// NewAssignHandler 创建主机分配操作处理器。将平台级或租户级主机分配给下层使用。
// +openapi:action=assign
// +openapi:resource=Host
// +openapi:summary=分配主机到租户或项目
func NewAssignHandler(hostStore HostStore, assignStore HostAssignmentStore) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		hostIDStr := params["hostId"]
		hostID, err := parseID(hostIDStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", hostIDStr), nil)
		}

		var req AssignRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, apierrors.NewBadRequest("invalid request body", nil)
		}

		if errs := ValidateAssignRequest(&req); errs.HasErrors() {
			return nil, apierrors.NewBadRequest("validation failed", errs)
		}

		// Check the host's scope: namespace hosts cannot be assigned
		host, err := hostStore.GetByID(ctx, hostID)
		if err != nil {
			return nil, err
		}
		if host.Scope == ScopeNamespace {
			return nil, apierrors.NewBadRequest("namespace-scoped hosts cannot be assigned", nil)
		}

		// Workspace hosts can only be assigned to namespaces in the same workspace
		if host.Scope == ScopeWorkspace && req.NamespaceID != "" {
			// Allowed: workspace host → namespace assignment
		} else if host.Scope == ScopeWorkspace && req.WorkspaceID != "" {
			return nil, apierrors.NewBadRequest("workspace-scoped hosts cannot be assigned to another workspace", nil)
		}

		var wsID, nsID *int64
		if req.WorkspaceID != "" {
			wid, err := parseID(req.WorkspaceID)
			if err != nil {
				return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspaceId: %s", req.WorkspaceID), nil)
			}
			wsID = &wid
		}
		if req.NamespaceID != "" {
			nid, err := parseID(req.NamespaceID)
			if err != nil {
				return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespaceId: %s", req.NamespaceID), nil)
			}
			nsID = &nid
		}

		_, err = assignStore.Assign(ctx, hostID, wsID, nsID)
		if err != nil {
			return nil, err
		}

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "host assigned successfully",
		}, nil
	}
}

// ===== unassign 主机取消分配操作 =====

// NewUnassignHandler 创建主机取消分配操作处理器。
// +openapi:action=unassign
// +openapi:resource=Host
// +openapi:summary=取消主机分配
func NewUnassignHandler(assignStore HostAssignmentStore) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		hostIDStr := params["hostId"]
		hostID, err := parseID(hostIDStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", hostIDStr), nil)
		}

		var req AssignRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, apierrors.NewBadRequest("invalid request body", nil)
		}

		if errs := ValidateAssignRequest(&req); errs.HasErrors() {
			return nil, apierrors.NewBadRequest("validation failed", errs)
		}

		if req.WorkspaceID != "" {
			wsID, err := parseID(req.WorkspaceID)
			if err != nil {
				return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspaceId: %s", req.WorkspaceID), nil)
			}
			if err := assignStore.UnassignWorkspace(ctx, hostID, wsID); err != nil {
				return nil, err
			}
		}

		if req.NamespaceID != "" {
			nsID, err := parseID(req.NamespaceID)
			if err != nil {
				return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespaceId: %s", req.NamespaceID), nil)
			}
			if err := assignStore.UnassignNamespace(ctx, hostID, nsID); err != nil {
				return nil, err
			}
		}

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "host unassigned successfully",
		}, nil
	}
}

// ===== bind-environment 绑定环境操作 =====

// NewBindEnvironmentHandler 创建主机绑定环境操作处理器。
// +openapi:action=bind-environment
// +openapi:resource=Host
// +openapi:summary=绑定主机到环境
func NewBindEnvironmentHandler(hostStore HostStore) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		hostIDStr := params["hostId"]
		hostID, err := parseID(hostIDStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", hostIDStr), nil)
		}

		var req BindEnvironmentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, apierrors.NewBadRequest("invalid request body", nil)
		}

		if errs := ValidateBindEnvironmentRequest(&req); errs.HasErrors() {
			return nil, apierrors.NewBadRequest("validation failed", errs)
		}

		envID, err := parseID(req.EnvironmentID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environmentId: %s", req.EnvironmentID), nil)
		}

		if err := hostStore.BindEnvironment(ctx, hostID, envID); err != nil {
			return nil, err
		}

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "host bound to environment successfully",
		}, nil
	}
}

// ===== unbind-environment 解绑环境操作 =====

// NewUnbindEnvironmentHandler 创建主机解绑环境操作处理器。
// +openapi:action=unbind-environment
// +openapi:resource=Host
// +openapi:summary=解绑主机环境
func NewUnbindEnvironmentHandler(hostStore HostStore) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		hostIDStr := params["hostId"]
		hostID, err := parseID(hostIDStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", hostIDStr), nil)
		}

		if err := hostStore.UnbindEnvironment(ctx, hostID); err != nil {
			return nil, err
		}

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "host unbound from environment successfully",
		}, nil
	}
}

// ===== hostAssignmentsVerbStorage 主机分配记录视图 =====

// hostAssignmentsVerbStorage 主机分配记录的 custom verb 存储。
// 注册为 GET /hosts/{hostId}:assignments
type hostAssignmentsVerbStorage struct {
	assignStore HostAssignmentStore
}

// NewHostAssignmentsVerb 创建主机分配记录视图存储。
// +openapi:customverb=assignments
// +openapi:resource=Host
// +openapi:response=HostAssignmentList
// +openapi:summary=获取主机的分配记录列表
func NewHostAssignmentsVerb(assignStore HostAssignmentStore) rest.Lister {
	return &hostAssignmentsVerbStorage{assignStore: assignStore}
}

func (s *hostAssignmentsVerbStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	hostID, err := parseID(options.PathParams["hostId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid host ID: %s", options.PathParams["hostId"]), nil)
	}

	rows, err := s.assignStore.ListByHostID(ctx, hostID)
	if err != nil {
		return nil, err
	}

	items := make([]HostAssignment, len(rows))
	for i, row := range rows {
		items[i] = assignmentRowToAPI(&row)
	}

	return &HostAssignmentList{
		TypeMeta: runtime.TypeMeta{Kind: "HostAssignmentList"},
		Items:    items,
	}, nil
}

// ===== envHostsVerbStorage 环境下主机视图 =====

// envHostsVerbStorage 环境关联主机的 custom verb 存储。
// 注册为 GET /environments/{environmentId}:hosts
type envHostsVerbStorage struct {
	envStore EnvironmentStore
}

// NewEnvHostsVerb 创建环境下主机列表视图存储。
// +openapi:customverb=hosts
// +openapi:resource=Environment
// +openapi:response=HostList
// +openapi:summary=获取环境下的主机列表
func NewEnvHostsVerb(envStore EnvironmentStore) rest.Lister {
	return &envHostsVerbStorage{envStore: envStore}
}

func (s *envHostsVerbStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	envID, err := parseID(options.PathParams["environmentId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid environment ID: %s", options.PathParams["environmentId"]), nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.envStore.ListHostsByEnvID(ctx, envID, query)
	if err != nil {
		return nil, err
	}

	items := make([]Host, len(result.Items))
	for i, item := range result.Items {
		items[i] = hostByEnvRowToAPI(&item)
	}

	return &HostList{
		TypeMeta:   runtime.TypeMeta{Kind: "HostList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// ===== helpers =====

// restOptionsToListQuery converts REST ListOptions to a db.ListQuery.
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

// hostSpecToPatchFields extracts non-zero fields from a Host for patch operations.
func hostSpecToPatchFields(host *Host) map[string]any {
	fields := make(map[string]any)
	if host.ObjectMeta.Name != "" {
		fields["name"] = host.ObjectMeta.Name
	}
	if host.Spec.Hostname != "" {
		fields["hostname"] = host.Spec.Hostname
	}
	if host.Spec.IPAddress != "" {
		fields["ip_address"] = host.Spec.IPAddress
	}
	if host.Spec.OS != "" {
		fields["os"] = host.Spec.OS
	}
	if host.Spec.Arch != "" {
		fields["arch"] = host.Spec.Arch
	}
	if host.Spec.CPUCores != 0 {
		fields["cpu_cores"] = host.Spec.CPUCores
	}
	if host.Spec.MemoryMB != 0 {
		fields["memory_mb"] = host.Spec.MemoryMB
	}
	if host.Spec.DiskGB != 0 {
		fields["disk_gb"] = host.Spec.DiskGB
	}
	if host.Spec.Labels != nil {
		raw, _ := json.Marshal(host.Spec.Labels)
		fields["labels"] = json.RawMessage(raw)
	}
	if host.Spec.Status != "" {
		fields["status"] = host.Spec.Status
	}
	return fields
}

// envSpecToPatchFields extracts non-zero fields from an Environment for patch operations.
func envSpecToPatchFields(env *Environment) map[string]any {
	fields := make(map[string]any)
	if env.ObjectMeta.Name != "" {
		fields["name"] = env.ObjectMeta.Name
	}
	if env.Spec.DisplayName != "" {
		fields["display_name"] = env.Spec.DisplayName
	}
	if env.Spec.Description != "" {
		fields["description"] = env.Spec.Description
	}
	if env.Spec.EnvType != "" {
		fields["env_type"] = env.Spec.EnvType
	}
	if env.Spec.Status != "" {
		fields["status"] = env.Spec.Status
	}
	return fields
}

// ===== DB → API conversion helpers =====

// hostToAPI converts a DBHost to an API Host.
func hostToAPI(h *DBHost) *Host {
	return &Host{
		TypeMeta: runtime.TypeMeta{Kind: "Host"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(h.ID, 10),
			Name:      h.Name,
			CreatedAt: new(h.CreatedAt),
			UpdatedAt: new(h.UpdatedAt),
		},
		Spec: HostSpec{
			Hostname:      h.Hostname,
			IPAddress:     h.IpAddress,
			OS:            h.Os,
			Arch:          h.Arch,
			CPUCores:      h.CpuCores,
			MemoryMB:      h.MemoryMb,
			DiskGB:        h.DiskGb,
			Labels:        labelsToMap(h.Labels),
			Scope:         h.Scope,
			WorkspaceID:   optionalIDToStr(h.WorkspaceID),
			NamespaceID:   optionalIDToStr(h.NamespaceID),
			EnvironmentID: optionalIDToStr(h.EnvironmentID),
			Status:        h.Status,
		},
	}
}

// hostWithEnvToAPI converts a DBHostWithEnv (GetHostByIDRow) to an API Host.
func hostWithEnvToAPI(h *DBHostWithEnv) *Host {
	host := &Host{
		TypeMeta: runtime.TypeMeta{Kind: "Host"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(h.ID, 10),
			Name:      h.Name,
			CreatedAt: new(h.CreatedAt),
			UpdatedAt: new(h.UpdatedAt),
		},
		Spec: HostSpec{
			Hostname:      h.Hostname,
			IPAddress:     h.IpAddress,
			OS:            h.Os,
			Arch:          h.Arch,
			CPUCores:      h.CpuCores,
			MemoryMB:      h.MemoryMb,
			DiskGB:        h.DiskGb,
			Labels:        labelsToMap(h.Labels),
			Scope:         h.Scope,
			WorkspaceID:   optionalIDToStr(h.WorkspaceID),
			NamespaceID:   optionalIDToStr(h.NamespaceID),
			EnvironmentID: optionalIDToStr(h.EnvironmentID),
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	return host
}

// hostPlatformRowToAPI converts a DBHostPlatformRow (ListHostsPlatformRow) to an API Host.
func hostPlatformRowToAPI(h *DBHostPlatformRow) Host {
	host := Host{
		TypeMeta: runtime.TypeMeta{Kind: "Host"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(h.ID, 10),
			Name:      h.Name,
			CreatedAt: new(h.CreatedAt),
			UpdatedAt: new(h.UpdatedAt),
		},
		Spec: HostSpec{
			Hostname:      h.Hostname,
			IPAddress:     h.IpAddress,
			OS:            h.Os,
			Arch:          h.Arch,
			CPUCores:      h.CpuCores,
			MemoryMB:      h.MemoryMb,
			DiskGB:        h.DiskGb,
			Labels:        labelsToMap(h.Labels),
			Scope:         h.Scope,
			WorkspaceID:   optionalIDToStr(h.WorkspaceID),
			NamespaceID:   optionalIDToStr(h.NamespaceID),
			EnvironmentID: optionalIDToStr(h.EnvironmentID),
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	return host
}

// hostWorkspaceRowToAPI converts a DBHostWorkspaceRow to an API Host (includes Origin).
func hostWorkspaceRowToAPI(h *DBHostWorkspaceRow) Host {
	host := Host{
		TypeMeta: runtime.TypeMeta{Kind: "Host"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(h.ID, 10),
			Name:      h.Name,
			CreatedAt: new(h.CreatedAt),
			UpdatedAt: new(h.UpdatedAt),
		},
		Spec: HostSpec{
			Hostname:      h.Hostname,
			IPAddress:     h.IpAddress,
			OS:            h.Os,
			Arch:          h.Arch,
			CPUCores:      h.CpuCores,
			MemoryMB:      h.MemoryMb,
			DiskGB:        h.DiskGb,
			Labels:        labelsToMap(h.Labels),
			Scope:         h.Scope,
			WorkspaceID:   optionalIDToStr(h.WorkspaceID),
			NamespaceID:   optionalIDToStr(h.NamespaceID),
			EnvironmentID: optionalIDToStr(h.EnvironmentID),
			Origin:        h.Origin,
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	return host
}

// hostNamespaceRowToAPI converts a DBHostNamespaceRow to an API Host (includes Origin).
func hostNamespaceRowToAPI(h *DBHostNamespaceRow) Host {
	host := Host{
		TypeMeta: runtime.TypeMeta{Kind: "Host"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(h.ID, 10),
			Name:      h.Name,
			CreatedAt: new(h.CreatedAt),
			UpdatedAt: new(h.UpdatedAt),
		},
		Spec: HostSpec{
			Hostname:      h.Hostname,
			IPAddress:     h.IpAddress,
			OS:            h.Os,
			Arch:          h.Arch,
			CPUCores:      h.CpuCores,
			MemoryMB:      h.MemoryMb,
			DiskGB:        h.DiskGb,
			Labels:        labelsToMap(h.Labels),
			Scope:         h.Scope,
			WorkspaceID:   optionalIDToStr(h.WorkspaceID),
			NamespaceID:   optionalIDToStr(h.NamespaceID),
			EnvironmentID: optionalIDToStr(h.EnvironmentID),
			Origin:        h.Origin,
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	return host
}

// hostByEnvRowToAPI converts a DBHostByEnvRow to an API Host.
func hostByEnvRowToAPI(h *DBHostByEnvRow) Host {
	host := Host{
		TypeMeta: runtime.TypeMeta{Kind: "Host"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(h.ID, 10),
			Name:      h.Name,
			CreatedAt: new(h.CreatedAt),
			UpdatedAt: new(h.UpdatedAt),
		},
		Spec: HostSpec{
			Hostname:      h.Hostname,
			IPAddress:     h.IpAddress,
			OS:            h.Os,
			Arch:          h.Arch,
			CPUCores:      h.CpuCores,
			MemoryMB:      h.MemoryMb,
			DiskGB:        h.DiskGb,
			Labels:        labelsToMap(h.Labels),
			Scope:         h.Scope,
			WorkspaceID:   optionalIDToStr(h.WorkspaceID),
			NamespaceID:   optionalIDToStr(h.NamespaceID),
			EnvironmentID: optionalIDToStr(h.EnvironmentID),
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	return host
}

// envToAPI converts a DBEnvironment to an API Environment.
func envToAPI(e *DBEnvironment) *Environment {
	return &Environment{
		TypeMeta: runtime.TypeMeta{Kind: "Environment"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(e.ID, 10),
			Name:      e.Name,
			CreatedAt: new(e.CreatedAt),
			UpdatedAt: new(e.UpdatedAt),
		},
		Spec: EnvironmentSpec{
			DisplayName: e.DisplayName,
			Description: e.Description,
			EnvType:     e.EnvType,
			Scope:       e.Scope,
			WorkspaceID: optionalIDToStr(e.WorkspaceID),
			NamespaceID: optionalIDToStr(e.NamespaceID),
			Status:      e.Status,
		},
	}
}

// envWithCountsToAPI converts a DBEnvWithCounts to an API Environment (includes HostCount).
func envWithCountsToAPI(e *DBEnvWithCounts) *Environment {
	env := envToAPI(&DBEnvironment{
		ID:          e.ID,
		Name:        e.Name,
		DisplayName: e.DisplayName,
		Description: e.Description,
		EnvType:     e.EnvType,
		Scope:       e.Scope,
		WorkspaceID: e.WorkspaceID,
		NamespaceID: e.NamespaceID,
		Status:      e.Status,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	})
	env.Spec.HostCount = e.HostCount
	return env
}

// envPlatformRowToAPI converts a DBEnvPlatformRow to an API Environment.
func envPlatformRowToAPI(e *DBEnvPlatformRow) Environment {
	return Environment{
		TypeMeta: runtime.TypeMeta{Kind: "Environment"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(e.ID, 10),
			Name:      e.Name,
			CreatedAt: new(e.CreatedAt),
			UpdatedAt: new(e.UpdatedAt),
		},
		Spec: EnvironmentSpec{
			DisplayName: e.DisplayName,
			Description: e.Description,
			EnvType:     e.EnvType,
			Scope:       e.Scope,
			WorkspaceID: optionalIDToStr(e.WorkspaceID),
			NamespaceID: optionalIDToStr(e.NamespaceID),
			HostCount:   e.HostCount,
			Status:      e.Status,
		},
	}
}

// envWorkspaceRowToAPI converts a DBEnvWorkspaceRow to an API Environment.
func envWorkspaceRowToAPI(e *DBEnvWorkspaceRow) Environment {
	return Environment{
		TypeMeta: runtime.TypeMeta{Kind: "Environment"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(e.ID, 10),
			Name:      e.Name,
			CreatedAt: new(e.CreatedAt),
			UpdatedAt: new(e.UpdatedAt),
		},
		Spec: EnvironmentSpec{
			DisplayName: e.DisplayName,
			Description: e.Description,
			EnvType:     e.EnvType,
			Scope:       e.Scope,
			WorkspaceID: optionalIDToStr(e.WorkspaceID),
			NamespaceID: optionalIDToStr(e.NamespaceID),
			HostCount:   e.HostCount,
			Status:      e.Status,
		},
	}
}

// envNamespaceRowToAPI converts a DBEnvNamespaceRow to an API Environment.
func envNamespaceRowToAPI(e *DBEnvNamespaceRow) Environment {
	return Environment{
		TypeMeta: runtime.TypeMeta{Kind: "Environment"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(e.ID, 10),
			Name:      e.Name,
			CreatedAt: new(e.CreatedAt),
			UpdatedAt: new(e.UpdatedAt),
		},
		Spec: EnvironmentSpec{
			DisplayName: e.DisplayName,
			Description: e.Description,
			EnvType:     e.EnvType,
			Scope:       e.Scope,
			WorkspaceID: optionalIDToStr(e.WorkspaceID),
			NamespaceID: optionalIDToStr(e.NamespaceID),
			HostCount:   e.HostCount,
			Status:      e.Status,
		},
	}
}

// assignmentRowToAPI converts a DBAssignmentRow to an API HostAssignment.
func assignmentRowToAPI(a *DBAssignmentRow) HostAssignment {
	ha := HostAssignment{
		TypeMeta: runtime.TypeMeta{Kind: "HostAssignment"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(a.ID, 10),
			CreatedAt: new(a.CreatedAt),
		},
		Spec: HostAssignmentSpec{
			HostID:   strconv.FormatInt(a.HostID, 10),
			HostName: a.HostName,
		},
	}
	if a.WorkspaceID != nil {
		ha.Spec.WorkspaceID = strconv.FormatInt(*a.WorkspaceID, 10)
	}
	if a.WorkspaceName != nil {
		ha.Spec.WorkspaceName = *a.WorkspaceName
	}
	if a.NamespaceID != nil {
		ha.Spec.NamespaceID = strconv.FormatInt(*a.NamespaceID, 10)
	}
	if a.NamespaceName != nil {
		ha.Spec.NamespaceName = *a.NamespaceName
	}
	return ha
}

// optionalIDToStr converts a *int64 to a string, returning empty string if nil.
func optionalIDToStr(id *int64) string {
	if id == nil {
		return ""
	}
	return strconv.FormatInt(*id, 10)
}

// labelsToMap converts a json.RawMessage to a map[string]string.
func labelsToMap(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

var parseID = rest.ParseID
