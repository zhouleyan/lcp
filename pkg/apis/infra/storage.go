package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/ipam"
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

	dbIPs, err := convertIPConfigs(host.Spec.IPs)
	if err != nil {
		return nil, err
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
	}, dbIPs)
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

	dbIPs, err := convertIPConfigs(host.Spec.IPs)
	if err != nil {
		return nil, err
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
	}, dbIPs)
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

	dbIPs, err := convertIPConfigs(host.Spec.IPs)
	if err != nil {
		return nil, err
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
	}, dbIPs)
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

// ===== Host IP sub-resource storage =====

// hostIPStorage 主机 IP 子资源的 REST 存储实现，支持列表、添加和移除。
type hostIPStorage struct {
	hostStore HostStore
	ipBinder  IPBinder
}

// NewHostIPStorage 创建主机 IP 子资源 REST 存储。
func NewHostIPStorage(hostStore HostStore, ipBinder IPBinder) *hostIPStorage {
	return &hostIPStorage{hostStore: hostStore, ipBinder: ipBinder}
}

func (s *hostIPStorage) NewObject() runtime.Object { return &IPConfig{} }

// Create 为主机追加 IP。
// +openapi:summary=为主机追加 IP
func (s *hostIPStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	cfg, ok := obj.(*IPConfig)
	if !ok {
		return nil, fmt.Errorf("expected *IPConfig, got %T", obj)
	}

	hostID, err := parseID(options.PathParams["hostId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid host ID", nil)
	}

	subnetID, err := parseID(cfg.SubnetID)
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid subnet ID", nil)
	}

	if err := s.hostStore.AddIP(ctx, hostID, DBIPConfig{SubnetID: subnetID, IP: cfg.IP}); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Delete 从主机移除 IP（解绑但不释放）。
// +openapi:summary=从主机移除 IP
func (s *hostIPStorage) Delete(ctx context.Context, options *rest.DeleteOptions) (runtime.Object, error) {
	hostID, err := parseID(options.PathParams["hostId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid host ID", nil)
	}

	allocID, err := parseID(options.PathParams["ipId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid IP allocation ID", nil)
	}

	if err := s.ipBinder.UnbindIPAllocationFromHost(ctx, allocID, hostID); err != nil {
		return nil, err
	}

	return nil, nil
}

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
	delete(query.Filters, "inherit")

	if options.Filters["inherit"] == "true" {
		result, err := s.envStore.ListByWorkspaceIDInherit(ctx, wsID, query)
		if err != nil {
			return nil, err
		}
		items := make([]Environment, len(result.Items))
		for i, item := range result.Items {
			items[i] = envWorkspaceInheritRowToAPI(&item)
		}
		return &EnvironmentList{
			TypeMeta:   runtime.TypeMeta{Kind: "EnvironmentList"},
			Items:      items,
			TotalCount: result.TotalCount,
		}, nil
	}

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
	delete(query.Filters, "inherit")

	if options.Filters["inherit"] == "true" {
		result, err := s.envStore.ListByNamespaceIDInherit(ctx, nsID, query)
		if err != nil {
			return nil, err
		}
		items := make([]Environment, len(result.Items))
		for i, item := range result.Items {
			items[i] = envNamespaceInheritRowToAPI(&item)
		}
		return &EnvironmentList{
			TypeMeta:   runtime.TypeMeta{Kind: "EnvironmentList"},
			Items:      items,
			TotalCount: result.TotalCount,
		}, nil
	}

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

// ===== bind-environment 绑定环境操作 =====

// NewBindEnvironmentHandler 创建主机绑定环境操作处理器。
// +openapi:action=bind-environment
// +openapi:resource=Host
// +openapi:summary=绑定主机到环境
func NewBindEnvironmentHandler(hostStore HostStore, envStore EnvironmentStore) rest.HandlerFunc {
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

		// Validate environment scope is compatible with host scope.
		// Environment must be at the same level or higher than the host in the
		// scope hierarchy (platform → workspace → namespace).
		env, err := envStore.GetByID(ctx, envID)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, hostID)
		if err != nil {
			return nil, err
		}
		var nsParentWsID int64
		if host.Scope == "namespace" && host.NamespaceID != nil && env.Scope == "workspace" {
			nsParentWsID, err = hostStore.GetWorkspaceIDByNamespaceID(ctx, *host.NamespaceID)
			if err != nil {
				return nil, err
			}
		}
		if !isEnvScopeCompatible(host, env, nsParentWsID) {
			return nil, apierrors.NewBadRequest("environment scope is not compatible with host scope", nil)
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

// ===== regionStorage 平台级区域存储 =====

// regionStorage 平台级区域资源的 REST 存储实现，支持 CRUD 和批量删除。
type regionStorage struct {
	regionStore RegionStore
}

// NewRegionStorage 创建平台级区域 REST 存储。
func NewRegionStorage(regionStore RegionStore) rest.StandardStorage {
	return &regionStorage{regionStore: regionStore}
}

func (s *regionStorage) NewObject() runtime.Object { return &Region{} }

// Get 获取区域详情。
// +openapi:summary=获取区域详情
func (s *regionStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["regionId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid region ID: %s", id), nil)
	}

	region, err := s.regionStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}

	return regionWithCountsToAPI(region), nil
}

// List 获取区域列表。
// +openapi:summary=获取区域列表
func (s *regionStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.regionStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Region, len(result.Items))
	for i, item := range result.Items {
		items[i] = regionRowToAPI(&item)
	}

	return &RegionList{
		TypeMeta:   runtime.TypeMeta{Kind: "RegionList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建区域。
// +openapi:summary=创建区域
func (s *regionStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	region, ok := obj.(*Region)
	if !ok {
		return nil, fmt.Errorf("expected *Region, got %T", obj)
	}

	if errs := ValidateRegionCreate(region.ObjectMeta.Name, &region.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return region, nil
	}

	status := region.Spec.Status
	if status == "" {
		status = "active"
	}

	created, err := s.regionStore.Create(ctx, &DBRegion{
		Name:        region.ObjectMeta.Name,
		DisplayName: region.Spec.DisplayName,
		Description: region.Spec.Description,
		Status:      status,
		Latitude:    region.Spec.Latitude,
		Longitude:   region.Spec.Longitude,
	})
	if err != nil {
		return nil, err
	}

	return regionToAPI(created), nil
}

// Update 全量更新区域信息。
// +openapi:summary=更新区域信息（全量）
func (s *regionStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	region, ok := obj.(*Region)
	if !ok {
		return nil, fmt.Errorf("expected *Region, got %T", obj)
	}

	if errs := ValidateRegionUpdate(&region.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return region, nil
	}

	id := options.PathParams["regionId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid region ID: %s", id), nil)
	}

	updated, err := s.regionStore.Update(ctx, &DBRegion{
		ID:          rid,
		Name:        region.ObjectMeta.Name,
		DisplayName: region.Spec.DisplayName,
		Description: region.Spec.Description,
		Status:      region.Spec.Status,
		Latitude:    region.Spec.Latitude,
		Longitude:   region.Spec.Longitude,
	})
	if err != nil {
		return nil, err
	}

	return regionToAPI(updated), nil
}

// Patch 部分更新区域信息。
// +openapi:summary=更新区域信息（部分）
func (s *regionStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	region, ok := obj.(*Region)
	if !ok {
		return nil, fmt.Errorf("expected *Region, got %T", obj)
	}

	id := options.PathParams["regionId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid region ID: %s", id), nil)
	}

	fields := regionSpecToPatchFields(region)

	patched, err := s.regionStore.Patch(ctx, rid, fields)
	if err != nil {
		return nil, err
	}

	return regionToAPI(patched), nil
}

// Delete 删除单个区域。
// +openapi:summary=删除区域
func (s *regionStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["regionId"]
	rid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid region ID: %s", id), nil)
	}

	return s.regionStore.Delete(ctx, rid)
}

// DeleteCollection 批量删除区域。
// +openapi:summary=批量删除区域
func (s *regionStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		rid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid region ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, rid)
	}

	count, err := s.regionStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== regionSiteStorage 区域下站点列表 =====

// regionSiteStorage 区域下站点资源的嵌套列表存储。
// +openapi:path=/regions/{regionId}/sites
type regionSiteStorage struct {
	siteStore SiteStore
}

// NewRegionSiteStorage 创建区域下站点列表存储。
func NewRegionSiteStorage(siteStore SiteStore) rest.Lister {
	return &regionSiteStorage{siteStore: siteStore}
}

// List 获取区域下的站点列表。
// +openapi:summary=获取数据中心列表
// +openapi:summary.regions.sites=获取区域下的数据中心列表
func (s *regionSiteStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	regionIDStr := options.PathParams["regionId"]
	if _, err := parseID(regionIDStr); err != nil {
		return nil, apierrors.NewBadRequest("invalid region ID", nil)
	}

	query := restOptionsToListQuery(options)
	if query.Filters == nil {
		query.Filters = map[string]any{}
	}
	query.Filters["regionId"] = regionIDStr

	result, err := s.siteStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Site, len(result.Items))
	for i, item := range result.Items {
		items[i] = siteRowToAPI(&item)
	}

	return &SiteList{
		TypeMeta:   runtime.TypeMeta{Kind: "SiteList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// ===== siteStorage 平台级站点存储 =====

// siteStorage 平台级站点资源的 REST 存储实现，支持 CRUD 和批量删除。
type siteStorage struct {
	siteStore SiteStore
}

// NewSiteStorage 创建平台级站点 REST 存储。
func NewSiteStorage(siteStore SiteStore) rest.StandardStorage {
	return &siteStorage{siteStore: siteStore}
}

func (s *siteStorage) NewObject() runtime.Object { return &Site{} }

// Get 获取站点详情。
// +openapi:summary=获取数据中心详情
func (s *siteStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["siteId"]
	sid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid site ID: %s", id), nil)
	}

	site, err := s.siteStore.GetByID(ctx, sid)
	if err != nil {
		return nil, err
	}

	return siteWithDetailsToAPI(site), nil
}

// List 获取数据中心列表。
// +openapi:summary=获取数据中心列表
func (s *siteStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.siteStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Site, len(result.Items))
	for i, item := range result.Items {
		items[i] = siteRowToAPI(&item)
	}

	return &SiteList{
		TypeMeta:   runtime.TypeMeta{Kind: "SiteList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建站点。
// +openapi:summary=创建数据中心
func (s *siteStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	site, ok := obj.(*Site)
	if !ok {
		return nil, fmt.Errorf("expected *Site, got %T", obj)
	}

	if errs := ValidateSiteCreate(site.ObjectMeta.Name, &site.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return site, nil
	}

	regionID, err := parseID(site.Spec.RegionID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid regionId: %s", site.Spec.RegionID), nil)
	}

	status := site.Spec.Status
	if status == "" {
		status = "active"
	}

	created, err := s.siteStore.Create(ctx, &DBSite{
		Name:         site.ObjectMeta.Name,
		DisplayName:  site.Spec.DisplayName,
		Description:  site.Spec.Description,
		RegionID:     regionID,
		Status:       status,
		Address:      site.Spec.Address,
		Latitude:     site.Spec.Latitude,
		Longitude:    site.Spec.Longitude,
		ContactName:  site.Spec.ContactName,
		ContactPhone: site.Spec.ContactPhone,
		ContactEmail: site.Spec.ContactEmail,
	})
	if err != nil {
		return nil, err
	}

	return siteToAPI(created), nil
}

// Update 全量更新站点信息。
// +openapi:summary=更新数据中心信息（全量）
func (s *siteStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	site, ok := obj.(*Site)
	if !ok {
		return nil, fmt.Errorf("expected *Site, got %T", obj)
	}

	if errs := ValidateSiteUpdate(&site.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return site, nil
	}

	id := options.PathParams["siteId"]
	sid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid site ID: %s", id), nil)
	}

	var regionID int64
	if site.Spec.RegionID != "" {
		regionID, err = parseID(site.Spec.RegionID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid regionId: %s", site.Spec.RegionID), nil)
		}
	}

	updated, err := s.siteStore.Update(ctx, &DBSite{
		ID:           sid,
		Name:         site.ObjectMeta.Name,
		DisplayName:  site.Spec.DisplayName,
		Description:  site.Spec.Description,
		RegionID:     regionID,
		Status:       site.Spec.Status,
		Address:      site.Spec.Address,
		Latitude:     site.Spec.Latitude,
		Longitude:    site.Spec.Longitude,
		ContactName:  site.Spec.ContactName,
		ContactPhone: site.Spec.ContactPhone,
		ContactEmail: site.Spec.ContactEmail,
	})
	if err != nil {
		return nil, err
	}

	return siteToAPI(updated), nil
}

// Patch 部分更新站点信息。
// +openapi:summary=更新数据中心信息（部分）
func (s *siteStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	site, ok := obj.(*Site)
	if !ok {
		return nil, fmt.Errorf("expected *Site, got %T", obj)
	}

	id := options.PathParams["siteId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	sid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid site ID: %s", id), nil)
	}

	fields := siteSpecToPatchFields(site)

	patched, err := s.siteStore.Patch(ctx, sid, fields)
	if err != nil {
		return nil, err
	}

	return siteToAPI(patched), nil
}

// Delete 删除单个站点。
// +openapi:summary=删除数据中心
func (s *siteStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["siteId"]
	sid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid site ID: %s", id), nil)
	}

	return s.siteStore.Delete(ctx, sid)
}

// DeleteCollection 批量删除站点。
// +openapi:summary=批量删除数据中心
func (s *siteStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		sid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid site ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, sid)
	}

	count, err := s.siteStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== siteLocationStorage 站点下机房列表 =====

// siteLocationStorage 站点下机房资源的嵌套列表存储。
// +openapi:path=/sites/{siteId}/locations
type siteLocationStorage struct {
	locationStore LocationStore
}

// NewSiteLocationStorage 创建站点下机房列表存储。
func NewSiteLocationStorage(locationStore LocationStore) rest.Lister {
	return &siteLocationStorage{locationStore: locationStore}
}

// List 获取站点下的机房列表。
// +openapi:summary=获取机房列表
// +openapi:summary.sites.locations=获取数据中心下的机房列表
func (s *siteLocationStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	siteIDStr := options.PathParams["siteId"]
	if _, err := parseID(siteIDStr); err != nil {
		return nil, apierrors.NewBadRequest("invalid site ID", nil)
	}

	query := restOptionsToListQuery(options)
	if query.Filters == nil {
		query.Filters = map[string]any{}
	}
	query.Filters["siteId"] = siteIDStr

	result, err := s.locationStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Location, len(result.Items))
	for i, item := range result.Items {
		items[i] = locationRowToAPI(&item)
	}

	return &LocationList{
		TypeMeta:   runtime.TypeMeta{Kind: "LocationList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// ===== locationRackStorage 机房下机柜列表 =====

// locationRackStorage 机房下机柜资源的嵌套列表存储。
// +openapi:path=/locations/{locationId}/racks
type locationRackStorage struct {
	rackStore RackStore
}

// NewLocationRackStorage 创建机房下机柜列表存储。
func NewLocationRackStorage(rackStore RackStore) rest.Lister {
	return &locationRackStorage{rackStore: rackStore}
}

// List 获取机房下的机柜列表。
// +openapi:summary=获取机柜列表
// +openapi:summary.locations.racks=获取机房下的机柜列表
func (s *locationRackStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	locationIDStr := options.PathParams["locationId"]
	if _, err := parseID(locationIDStr); err != nil {
		return nil, apierrors.NewBadRequest("invalid location ID", nil)
	}

	query := restOptionsToListQuery(options)
	if query.Filters == nil {
		query.Filters = map[string]any{}
	}
	query.Filters["locationId"] = locationIDStr

	result, err := s.rackStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Rack, len(result.Items))
	for i, item := range result.Items {
		items[i] = rackRowToAPI(&item)
	}

	return &RackList{
		TypeMeta:   runtime.TypeMeta{Kind: "RackList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// ===== rackStorage 平台级机柜存储 =====

// rackStorage 平台级机柜资源的 REST 存储实现，支持 CRUD 和批量删除。
type rackStorage struct {
	rackStore RackStore
}

// NewRackStorage 创建平台级机柜 REST 存储。
func NewRackStorage(rackStore RackStore) rest.StandardStorage {
	return &rackStorage{rackStore: rackStore}
}

func (s *rackStorage) NewObject() runtime.Object { return &Rack{} }

// Get 获取机柜详情。
// +openapi:summary=获取机柜详情
func (s *rackStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["rackId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid rack ID: %s", id), nil)
	}

	rack, err := s.rackStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}

	return rackWithDetailsToAPI(rack), nil
}

// List 获取机柜列表。
// +openapi:summary=获取机柜列表
func (s *rackStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.rackStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Rack, len(result.Items))
	for i, item := range result.Items {
		items[i] = rackRowToAPI(&item)
	}

	return &RackList{
		TypeMeta:   runtime.TypeMeta{Kind: "RackList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建机柜。
// +openapi:summary=创建机柜
func (s *rackStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	rack, ok := obj.(*Rack)
	if !ok {
		return nil, fmt.Errorf("expected *Rack, got %T", obj)
	}

	if errs := ValidateRackCreate(rack.ObjectMeta.Name, &rack.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return rack, nil
	}

	locationID, err := parseID(rack.Spec.LocationID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid locationId: %s", rack.Spec.LocationID), nil)
	}

	status := rack.Spec.Status
	if status == "" {
		status = "active"
	}

	created, err := s.rackStore.Create(ctx, &DBRack{
		Name:          rack.ObjectMeta.Name,
		DisplayName:   rack.Spec.DisplayName,
		Description:   rack.Spec.Description,
		LocationID:    locationID,
		Status:        status,
		UHeight:       rack.Spec.UHeight,
		Position:      rack.Spec.Position,
		PowerCapacity: rack.Spec.PowerCapacity,
	})
	if err != nil {
		return nil, err
	}

	return rackToAPI(created), nil
}

// Update 全量更新机柜信息。
// +openapi:summary=更新机柜信息（全量）
func (s *rackStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	rack, ok := obj.(*Rack)
	if !ok {
		return nil, fmt.Errorf("expected *Rack, got %T", obj)
	}

	if errs := ValidateRackUpdate(&rack.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return rack, nil
	}

	id := options.PathParams["rackId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid rack ID: %s", id), nil)
	}

	var locationID int64
	if rack.Spec.LocationID != "" {
		locationID, err = parseID(rack.Spec.LocationID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid locationId: %s", rack.Spec.LocationID), nil)
		}
	}

	updated, err := s.rackStore.Update(ctx, &DBRack{
		ID:            rid,
		Name:          rack.ObjectMeta.Name,
		DisplayName:   rack.Spec.DisplayName,
		Description:   rack.Spec.Description,
		LocationID:    locationID,
		Status:        rack.Spec.Status,
		UHeight:       rack.Spec.UHeight,
		Position:      rack.Spec.Position,
		PowerCapacity: rack.Spec.PowerCapacity,
	})
	if err != nil {
		return nil, err
	}

	return rackToAPI(updated), nil
}

// Patch 部分更新机柜信息。
// +openapi:summary=更新机柜信息（部分）
func (s *rackStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	rack, ok := obj.(*Rack)
	if !ok {
		return nil, fmt.Errorf("expected *Rack, got %T", obj)
	}

	id := options.PathParams["rackId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid rack ID: %s", id), nil)
	}

	fields := rackSpecToPatchFields(rack)

	patched, err := s.rackStore.Patch(ctx, rid, fields)
	if err != nil {
		return nil, err
	}

	return rackToAPI(patched), nil
}

// Delete 删除单个机柜。
// +openapi:summary=删除机柜
func (s *rackStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["rackId"]
	rid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid rack ID: %s", id), nil)
	}

	return s.rackStore.Delete(ctx, rid)
}

// DeleteCollection 批量删除机柜。
// +openapi:summary=批量删除机柜
func (s *rackStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		rid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid rack ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, rid)
	}

	count, err := s.rackStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== locationStorage 平台级机房存储 =====

// locationStorage 平台级机房资源的 REST 存储实现，支持 CRUD 和批量删除。
type locationStorage struct {
	locationStore LocationStore
}

// NewLocationStorage 创建平台级机房 REST 存储。
func NewLocationStorage(locationStore LocationStore) rest.StandardStorage {
	return &locationStorage{locationStore: locationStore}
}

func (s *locationStorage) NewObject() runtime.Object { return &Location{} }

// Get 获取机房详情。
// +openapi:summary=获取机房详情
func (s *locationStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["locationId"]
	lid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid location ID: %s", id), nil)
	}

	location, err := s.locationStore.GetByID(ctx, lid)
	if err != nil {
		return nil, err
	}

	return locationWithDetailsToAPI(location), nil
}

// List 获取机房列表。
// +openapi:summary=获取机房列表
func (s *locationStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.locationStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Location, len(result.Items))
	for i, item := range result.Items {
		items[i] = locationRowToAPI(&item)
	}

	return &LocationList{
		TypeMeta:   runtime.TypeMeta{Kind: "LocationList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建机房。
// +openapi:summary=创建机房
func (s *locationStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	location, ok := obj.(*Location)
	if !ok {
		return nil, fmt.Errorf("expected *Location, got %T", obj)
	}

	if errs := ValidateLocationCreate(location.ObjectMeta.Name, &location.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return location, nil
	}

	siteID, err := parseID(location.Spec.SiteID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid siteId: %s", location.Spec.SiteID), nil)
	}

	status := location.Spec.Status
	if status == "" {
		status = "active"
	}

	created, err := s.locationStore.Create(ctx, &DBLocation{
		Name:         location.ObjectMeta.Name,
		DisplayName:  location.Spec.DisplayName,
		Description:  location.Spec.Description,
		SiteID:       siteID,
		Status:       status,
		Floor:        location.Spec.Floor,
		RackCapacity: location.Spec.RackCapacity,
		ContactName:  location.Spec.ContactName,
		ContactPhone: location.Spec.ContactPhone,
		ContactEmail: location.Spec.ContactEmail,
	})
	if err != nil {
		return nil, err
	}

	return locationToAPI(created), nil
}

// Update 全量更新机房信息。
// +openapi:summary=更新机房信息（全量）
func (s *locationStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	location, ok := obj.(*Location)
	if !ok {
		return nil, fmt.Errorf("expected *Location, got %T", obj)
	}

	if errs := ValidateLocationUpdate(&location.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return location, nil
	}

	id := options.PathParams["locationId"]
	lid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid location ID: %s", id), nil)
	}

	var siteID int64
	if location.Spec.SiteID != "" {
		siteID, err = parseID(location.Spec.SiteID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid siteId: %s", location.Spec.SiteID), nil)
		}
	}

	updated, err := s.locationStore.Update(ctx, &DBLocation{
		ID:           lid,
		Name:         location.ObjectMeta.Name,
		DisplayName:  location.Spec.DisplayName,
		Description:  location.Spec.Description,
		SiteID:       siteID,
		Status:       location.Spec.Status,
		Floor:        location.Spec.Floor,
		RackCapacity: location.Spec.RackCapacity,
		ContactName:  location.Spec.ContactName,
		ContactPhone: location.Spec.ContactPhone,
		ContactEmail: location.Spec.ContactEmail,
	})
	if err != nil {
		return nil, err
	}

	return locationToAPI(updated), nil
}

// Patch 部分更新机房信息。
// +openapi:summary=更新机房信息（部分）
func (s *locationStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	location, ok := obj.(*Location)
	if !ok {
		return nil, fmt.Errorf("expected *Location, got %T", obj)
	}

	id := options.PathParams["locationId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	lid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid location ID: %s", id), nil)
	}

	fields := locationSpecToPatchFields(location)

	patched, err := s.locationStore.Patch(ctx, lid, fields)
	if err != nil {
		return nil, err
	}

	return locationToAPI(patched), nil
}

// Delete 删除单个机房。
// +openapi:summary=删除机房
func (s *locationStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["locationId"]
	lid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid location ID: %s", id), nil)
	}

	return s.locationStore.Delete(ctx, lid)
}

// DeleteCollection 批量删除机房。
// +openapi:summary=批量删除机房
func (s *locationStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		lid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid location ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, lid)
	}

	count, err := s.locationStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== availableNetworkStorage 可用网络列表（ACL，只读） =====

// availableNetworkStorage 主机 IP 分配时的网络查询接口，读取 network 模块数据（ACL 层）。
// 注册在三个层级：/networks（平台）、/workspaces/{id}/networks（租户）、/workspaces/{id}/namespaces/{id}/networks（项目）。
// 使用 PermissionTargets: ["infra:hosts:*"] 复用主机权限，不产生新权限。
//
// +openapi:path=/workspaces/{workspaceId}/networks
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/networks
type availableNetworkStorage struct {
	reader NetworkReader
}

// NewAvailableNetworkStorage creates a read-only Lister for available networks.
func NewAvailableNetworkStorage(reader NetworkReader) rest.Lister {
	return &availableNetworkStorage{reader: reader}
}

func (s *availableNetworkStorage) NewObject() runtime.Object { return &AvailableNetwork{} }

// List 获取可用网络列表（含子网摘要）。
// +openapi:summary=获取可用网络列表
// +openapi:summary.workspaces.networks=获取租户下的可用网络列表
// +openapi:summary.workspaces.namespaces.networks=获取项目下的可用网络列表
func (s *availableNetworkStorage) List(ctx context.Context, _ *rest.ListOptions) (runtime.Object, error) {
	// 1. Load all active networks
	networks, err := s.reader.ListActiveNetworks(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	if len(networks) == 0 {
		return &AvailableNetworkList{
			TypeMeta: runtime.TypeMeta{Kind: "AvailableNetworkList"},
			Items:    []AvailableNetwork{},
		}, nil
	}

	// 2. Collect network IDs and load all subnets in one query
	networkIDs := make([]int64, len(networks))
	for i, n := range networks {
		networkIDs[i] = n.ID
	}

	subnets, err := s.reader.ListSubnetsByNetworkIDs(ctx, networkIDs)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	// 3. Group subnets by network ID
	subnetsByNetwork := make(map[int64][]SubnetSummary)
	for _, sub := range subnets {
		summary := subnetToSummary(&sub)
		subnetsByNetwork[sub.NetworkID] = append(subnetsByNetwork[sub.NetworkID], summary)
	}

	// 4. Build response
	items := make([]AvailableNetwork, len(networks))
	for i, n := range networks {
		subs := subnetsByNetwork[n.ID]
		if subs == nil {
			subs = []SubnetSummary{}
		}
		items[i] = AvailableNetwork{
			TypeMeta: runtime.TypeMeta{Kind: "AvailableNetwork"},
			ObjectMeta: types.ObjectMeta{
				ID:        strconv.FormatInt(n.ID, 10),
				Name:      n.Name,
				CreatedAt: &n.CreatedAt,
				UpdatedAt: &n.UpdatedAt,
			},
			Spec: AvailableNetworkSpec{
				DisplayName: n.DisplayName,
				Description: n.Description,
				CIDR:        n.Cidr,
				IsPublic:    n.IsPublic,
				SubnetCount: n.SubnetCount,
				Subnets:     subs,
			},
		}
	}

	return &AvailableNetworkList{
		TypeMeta:   runtime.TypeMeta{Kind: "AvailableNetworkList"},
		Items:      items,
		TotalCount: int64(len(items)),
	}, nil
}

// subnetToSummary converts a DB subnet row to a SubnetSummary with IP usage stats from bitmap.
func subnetToSummary(s *DBSubnet) SubnetSummary {
	summary := SubnetSummary{
		ID:          strconv.FormatInt(s.ID, 10),
		Name:        s.Name,
		DisplayName: s.DisplayName,
		CIDR:        s.Cidr,
		Gateway:     s.Gateway,
	}

	// Calculate IP usage from bitmap
	_, cidrNet, err := net.ParseCIDR(s.Cidr)
	if err == nil {
		r, err := ipam.NewCIDRRange(cidrNet)
		if err == nil {
			if len(s.Bitmap) > 0 {
				_ = r.LoadFromBytes(s.Bitmap)
			}
			summary.FreeIPs = r.Free()
			summary.UsedIPs = r.Used()
			summary.TotalIPs = r.Free() + r.Used()
		}
	}

	return summary
}

// ===== helpers =====

// convertIPConfigs converts API IPConfig slice to DB IPConfig slice with parsed IDs.
func convertIPConfigs(ips []IPConfig) ([]DBIPConfig, error) {
	if len(ips) == 0 {
		return nil, nil
	}
	dbIPs := make([]DBIPConfig, 0, len(ips))
	for _, cfg := range ips {
		sid, err := strconv.ParseInt(cfg.SubnetID, 10, 64)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", cfg.SubnetID), nil)
		}
		dbIPs = append(dbIPs, DBIPConfig{SubnetID: sid, IP: cfg.IP})
	}
	return dbIPs, nil
}

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

// regionSpecToPatchFields extracts non-zero fields from a Region for patch operations.
func regionSpecToPatchFields(r *Region) map[string]any {
	fields := make(map[string]any)
	if r.ObjectMeta.Name != "" {
		fields["name"] = r.ObjectMeta.Name
	}
	if r.Spec.DisplayName != "" {
		fields["displayName"] = r.Spec.DisplayName
	}
	if r.Spec.Description != "" {
		fields["description"] = r.Spec.Description
	}
	if r.Spec.Status != "" {
		fields["status"] = r.Spec.Status
	}
	if r.Spec.Latitude != nil {
		fields["latitude"] = r.Spec.Latitude
	}
	if r.Spec.Longitude != nil {
		fields["longitude"] = r.Spec.Longitude
	}
	return fields
}

// siteSpecToPatchFields extracts non-zero fields from a Site for patch operations.
func siteSpecToPatchFields(s *Site) map[string]any {
	fields := make(map[string]any)
	if s.ObjectMeta.Name != "" {
		fields["name"] = s.ObjectMeta.Name
	}
	if s.Spec.DisplayName != "" {
		fields["displayName"] = s.Spec.DisplayName
	}
	if s.Spec.Description != "" {
		fields["description"] = s.Spec.Description
	}
	if s.Spec.RegionID != "" {
		if regionID, err := parseID(s.Spec.RegionID); err == nil {
			fields["regionId"] = regionID
		}
	}
	if s.Spec.Status != "" {
		fields["status"] = s.Spec.Status
	}
	if s.Spec.Address != "" {
		fields["address"] = s.Spec.Address
	}
	if s.Spec.Latitude != nil {
		fields["latitude"] = s.Spec.Latitude
	}
	if s.Spec.Longitude != nil {
		fields["longitude"] = s.Spec.Longitude
	}
	if s.Spec.ContactName != "" {
		fields["contactName"] = s.Spec.ContactName
	}
	if s.Spec.ContactPhone != "" {
		fields["contactPhone"] = s.Spec.ContactPhone
	}
	if s.Spec.ContactEmail != "" {
		fields["contactEmail"] = s.Spec.ContactEmail
	}
	return fields
}

// locationSpecToPatchFields extracts non-zero fields from a Location for patch operations.
func locationSpecToPatchFields(l *Location) map[string]any {
	fields := make(map[string]any)
	if l.ObjectMeta.Name != "" {
		fields["name"] = l.ObjectMeta.Name
	}
	if l.Spec.DisplayName != "" {
		fields["displayName"] = l.Spec.DisplayName
	}
	if l.Spec.Description != "" {
		fields["description"] = l.Spec.Description
	}
	if l.Spec.SiteID != "" {
		if siteID, err := parseID(l.Spec.SiteID); err == nil {
			fields["siteId"] = siteID
		}
	}
	if l.Spec.Status != "" {
		fields["status"] = l.Spec.Status
	}
	if l.Spec.Floor != "" {
		fields["floor"] = l.Spec.Floor
	}
	if l.Spec.RackCapacity != 0 {
		fields["rackCapacity"] = l.Spec.RackCapacity
	}
	if l.Spec.ContactName != "" {
		fields["contactName"] = l.Spec.ContactName
	}
	if l.Spec.ContactPhone != "" {
		fields["contactPhone"] = l.Spec.ContactPhone
	}
	if l.Spec.ContactEmail != "" {
		fields["contactEmail"] = l.Spec.ContactEmail
	}
	return fields
}

// ===== DB → API conversion helpers =====

// parseAllocatedIPs parses the allocated_ips JSON column from SQL queries.
// The JSON contains subnetId as a number, which needs to be converted to a string.
func parseAllocatedIPs(raw interface{}) []AllocatedIP {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var items []struct {
		IP         string  `json:"ip"`
		SubnetID   float64 `json:"subnetId"`
		SubnetName string  `json:"subnetName"`
		SubnetCIDR string  `json:"subnetCidr"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}
	if len(items) == 0 {
		return nil
	}
	result := make([]AllocatedIP, len(items))
	for i, item := range items {
		result[i] = AllocatedIP{
			IP:         item.IP,
			SubnetID:   strconv.FormatInt(int64(item.SubnetID), 10),
			SubnetName: item.SubnetName,
			SubnetCIDR: item.SubnetCIDR,
		}
	}
	return result
}

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
			AllocatedIPs:  parseAllocatedIPs(h.AllocatedIps),
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	if h.WorkspaceName != nil {
		host.Spec.WorkspaceName = *h.WorkspaceName
	}
	if h.NamespaceName != nil {
		host.Spec.NamespaceName = *h.NamespaceName
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
			AllocatedIPs:  parseAllocatedIPs(h.AllocatedIps),
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	if h.WorkspaceName != nil {
		host.Spec.WorkspaceName = *h.WorkspaceName
	}
	if h.NamespaceName != nil {
		host.Spec.NamespaceName = *h.NamespaceName
	}
	return host
}

// hostWorkspaceRowToAPI converts a DBHostWorkspaceRow to an API Host.
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
			AllocatedIPs:  parseAllocatedIPs(h.AllocatedIps),
			Status:        h.Status,
		},
	}
	if h.EnvironmentName != nil {
		host.Spec.EnvironmentName = *h.EnvironmentName
	}
	if h.NamespaceName != nil {
		host.Spec.NamespaceName = *h.NamespaceName
	}
	return host
}

// hostNamespaceRowToAPI converts a DBHostNamespaceRow to an API Host.
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
			AllocatedIPs:  parseAllocatedIPs(h.AllocatedIps),
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
			AllocatedIPs:  parseAllocatedIPs(h.AllocatedIps),
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

// envWorkspaceInheritRowToAPI converts a DBEnvWorkspaceInheritRow to an API Environment.
func envWorkspaceInheritRowToAPI(e *DBEnvWorkspaceInheritRow) Environment {
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

// envNamespaceInheritRowToAPI converts a DBEnvNamespaceInheritRow to an API Environment.
func envNamespaceInheritRowToAPI(e *DBEnvNamespaceInheritRow) Environment {
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

// isEnvScopeCompatible checks whether the environment scope is on the host's
// inheritance chain. The environment scope must be the same level or higher
// than the host scope, and share the same workspace/namespace lineage.
//
// nsParentWsID is the workspace_id of the host's namespace (only needed when
// host.Scope == "namespace" and env.Scope == "workspace"); pass 0 otherwise.
//
// Rules:
//   - platform env → compatible with any host
//   - workspace env → compatible with workspace host (same ws) or namespace host (parent ws matches)
//   - namespace env → compatible only with namespace host (same ns)
func isEnvScopeCompatible(host *DBHostWithEnv, env *DBEnvWithCounts, nsParentWsID int64) bool {
	switch env.Scope {
	case "platform":
		return true
	case "workspace":
		if env.WorkspaceID == nil {
			return false
		}
		switch host.Scope {
		case "workspace":
			return host.WorkspaceID != nil && *host.WorkspaceID == *env.WorkspaceID
		case "namespace":
			return nsParentWsID > 0 && nsParentWsID == *env.WorkspaceID
		default:
			return false
		}
	case "namespace":
		if env.NamespaceID == nil {
			return false
		}
		return host.Scope == "namespace" && host.NamespaceID != nil && *host.NamespaceID == *env.NamespaceID
	default:
		return false
	}
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

// ===== Region DB → API conversion helpers =====

// regionToAPI converts a DBRegion to an API Region.
func regionToAPI(r *DBRegion) *Region {
	return &Region{
		TypeMeta: runtime.TypeMeta{Kind: "Region"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(r.ID, 10),
			Name:      r.Name,
			CreatedAt: new(r.CreatedAt),
			UpdatedAt: new(r.UpdatedAt),
		},
		Spec: RegionSpec{
			DisplayName: r.DisplayName,
			Description: r.Description,
			Status:      r.Status,
			Latitude:    r.Latitude,
			Longitude:   r.Longitude,
		},
	}
}

// regionWithCountsToAPI converts a DBRegionWithCounts (GetRegionByIDRow) to an API Region.
func regionWithCountsToAPI(r *DBRegionWithCounts) *Region {
	region := regionToAPI(&DBRegion{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Status:      r.Status,
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	})
	region.Spec.SiteCount = r.SiteCount
	return region
}

// regionRowToAPI converts a DBRegionListRow (ListRegionsRow) to an API Region.
func regionRowToAPI(r *DBRegionListRow) Region {
	return Region{
		TypeMeta: runtime.TypeMeta{Kind: "Region"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(r.ID, 10),
			Name:      r.Name,
			CreatedAt: new(r.CreatedAt),
			UpdatedAt: new(r.UpdatedAt),
		},
		Spec: RegionSpec{
			DisplayName: r.DisplayName,
			Description: r.Description,
			Status:      r.Status,
			Latitude:    r.Latitude,
			Longitude:   r.Longitude,
			SiteCount:   r.SiteCount,
		},
	}
}

// ===== Site DB → API conversion helpers =====

// siteToAPI converts a DBSite to an API Site.
func siteToAPI(s *DBSite) *Site {
	return &Site{
		TypeMeta: runtime.TypeMeta{Kind: "Site"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(s.ID, 10),
			Name:      s.Name,
			CreatedAt: new(s.CreatedAt),
			UpdatedAt: new(s.UpdatedAt),
		},
		Spec: SiteSpec{
			DisplayName:  s.DisplayName,
			Description:  s.Description,
			RegionID:     strconv.FormatInt(s.RegionID, 10),
			Status:       s.Status,
			Address:      s.Address,
			Latitude:     s.Latitude,
			Longitude:    s.Longitude,
			ContactName:  s.ContactName,
			ContactPhone: s.ContactPhone,
			ContactEmail: s.ContactEmail,
		},
	}
}

// siteWithDetailsToAPI converts a DBSiteWithDetails (GetSiteByIDRow) to an API Site.
func siteWithDetailsToAPI(s *DBSiteWithDetails) *Site {
	site := siteToAPI(&DBSite{
		ID:           s.ID,
		Name:         s.Name,
		DisplayName:  s.DisplayName,
		Description:  s.Description,
		RegionID:     s.RegionID,
		Status:       s.Status,
		Address:      s.Address,
		Latitude:     s.Latitude,
		Longitude:    s.Longitude,
		ContactName:  s.ContactName,
		ContactPhone: s.ContactPhone,
		ContactEmail: s.ContactEmail,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	})
	site.Spec.RegionName = s.RegionName
	site.Spec.LocationCount = s.LocationCount
	return site
}

// siteRowToAPI converts a DBSiteListRow (ListSitesRow) to an API Site.
func siteRowToAPI(s *DBSiteListRow) Site {
	return Site{
		TypeMeta: runtime.TypeMeta{Kind: "Site"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(s.ID, 10),
			Name:      s.Name,
			CreatedAt: new(s.CreatedAt),
			UpdatedAt: new(s.UpdatedAt),
		},
		Spec: SiteSpec{
			DisplayName:   s.DisplayName,
			Description:   s.Description,
			RegionID:      strconv.FormatInt(s.RegionID, 10),
			RegionName:    s.RegionName,
			Status:        s.Status,
			Address:       s.Address,
			Latitude:      s.Latitude,
			Longitude:     s.Longitude,
			ContactName:   s.ContactName,
			ContactPhone:  s.ContactPhone,
			ContactEmail:  s.ContactEmail,
			LocationCount: s.LocationCount,
		},
	}
}

// ===== Location DB → API conversion helpers =====

// locationToAPI converts a DBLocation to an API Location.
func locationToAPI(l *DBLocation) *Location {
	return &Location{
		TypeMeta: runtime.TypeMeta{Kind: "Location"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(l.ID, 10),
			Name:      l.Name,
			CreatedAt: new(l.CreatedAt),
			UpdatedAt: new(l.UpdatedAt),
		},
		Spec: LocationSpec{
			DisplayName:  l.DisplayName,
			Description:  l.Description,
			SiteID:       strconv.FormatInt(l.SiteID, 10),
			Status:       l.Status,
			Floor:        l.Floor,
			RackCapacity: l.RackCapacity,
			ContactName:  l.ContactName,
			ContactPhone: l.ContactPhone,
			ContactEmail: l.ContactEmail,
		},
	}
}

// locationWithDetailsToAPI converts a DBLocationWithDetails (GetLocationByIDRow) to an API Location.
func locationWithDetailsToAPI(l *DBLocationWithDetails) *Location {
	location := locationToAPI(&DBLocation{
		ID:           l.ID,
		Name:         l.Name,
		DisplayName:  l.DisplayName,
		Description:  l.Description,
		SiteID:       l.SiteID,
		Status:       l.Status,
		Floor:        l.Floor,
		RackCapacity: l.RackCapacity,
		ContactName:  l.ContactName,
		ContactPhone: l.ContactPhone,
		ContactEmail: l.ContactEmail,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	})
	location.Spec.SiteName = l.SiteName
	location.Spec.RegionID = strconv.FormatInt(l.RegionID, 10)
	location.Spec.RegionName = l.RegionName
	location.Spec.RackCount = l.RackCount
	return location
}

// locationRowToAPI converts a DBLocationListRow (ListLocationsRow) to an API Location.
func locationRowToAPI(l *DBLocationListRow) Location {
	return Location{
		TypeMeta: runtime.TypeMeta{Kind: "Location"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(l.ID, 10),
			Name:      l.Name,
			CreatedAt: new(l.CreatedAt),
			UpdatedAt: new(l.UpdatedAt),
		},
		Spec: LocationSpec{
			DisplayName:  l.DisplayName,
			Description:  l.Description,
			SiteID:       strconv.FormatInt(l.SiteID, 10),
			SiteName:     l.SiteName,
			RegionID:     strconv.FormatInt(l.RegionID, 10),
			RegionName:   l.RegionName,
			Status:       l.Status,
			Floor:        l.Floor,
			RackCapacity: l.RackCapacity,
			RackCount:    l.RackCount,
			ContactName:  l.ContactName,
			ContactPhone: l.ContactPhone,
			ContactEmail: l.ContactEmail,
		},
	}
}

// rackSpecToPatchFields extracts non-zero fields from a Rack for patch operations.
func rackSpecToPatchFields(r *Rack) map[string]any {
	fields := make(map[string]any)
	if r.ObjectMeta.Name != "" {
		fields["name"] = r.ObjectMeta.Name
	}
	if r.Spec.DisplayName != "" {
		fields["displayName"] = r.Spec.DisplayName
	}
	if r.Spec.Description != "" {
		fields["description"] = r.Spec.Description
	}
	if r.Spec.LocationID != "" {
		if locationID, err := parseID(r.Spec.LocationID); err == nil {
			fields["locationId"] = locationID
		}
	}
	if r.Spec.Status != "" {
		fields["status"] = r.Spec.Status
	}
	if r.Spec.UHeight != 0 {
		fields["uHeight"] = r.Spec.UHeight
	}
	if r.Spec.Position != "" {
		fields["position"] = r.Spec.Position
	}
	if r.Spec.PowerCapacity != "" {
		fields["powerCapacity"] = r.Spec.PowerCapacity
	}
	return fields
}

// rackToAPI converts a DBRack to an API Rack.
func rackToAPI(r *DBRack) *Rack {
	return &Rack{
		TypeMeta: runtime.TypeMeta{Kind: "Rack"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(r.ID, 10),
			Name:      r.Name,
			CreatedAt: new(r.CreatedAt),
			UpdatedAt: new(r.UpdatedAt),
		},
		Spec: RackSpec{
			DisplayName:   r.DisplayName,
			Description:   r.Description,
			LocationID:    strconv.FormatInt(r.LocationID, 10),
			Status:        r.Status,
			UHeight:       r.UHeight,
			Position:      r.Position,
			PowerCapacity: r.PowerCapacity,
		},
	}
}

// rackWithDetailsToAPI converts a DBRackWithDetails (GetRackByIDRow) to an API Rack.
func rackWithDetailsToAPI(r *DBRackWithDetails) *Rack {
	rack := rackToAPI(&DBRack{
		ID:            r.ID,
		Name:          r.Name,
		DisplayName:   r.DisplayName,
		Description:   r.Description,
		LocationID:    r.LocationID,
		Status:        r.Status,
		UHeight:       r.UHeight,
		Position:      r.Position,
		PowerCapacity: r.PowerCapacity,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	})
	rack.Spec.LocationName = r.LocationName
	rack.Spec.SiteID = strconv.FormatInt(r.SiteID, 10)
	rack.Spec.SiteName = r.SiteName
	rack.Spec.RegionID = strconv.FormatInt(r.RegionID, 10)
	rack.Spec.RegionName = r.RegionName
	return rack
}

// rackRowToAPI converts a DBRackListRow (ListRacksRow) to an API Rack.
func rackRowToAPI(r *DBRackListRow) Rack {
	return Rack{
		TypeMeta: runtime.TypeMeta{Kind: "Rack"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(r.ID, 10),
			Name:      r.Name,
			CreatedAt: new(r.CreatedAt),
			UpdatedAt: new(r.UpdatedAt),
		},
		Spec: RackSpec{
			DisplayName:   r.DisplayName,
			Description:   r.Description,
			LocationID:    strconv.FormatInt(r.LocationID, 10),
			LocationName:  r.LocationName,
			SiteID:        strconv.FormatInt(r.SiteID, 10),
			SiteName:      r.SiteName,
			RegionID:      strconv.FormatInt(r.RegionID, 10),
			RegionName:    r.RegionName,
			Status:        r.Status,
			UHeight:       r.UHeight,
			Position:      r.Position,
			PowerCapacity: r.PowerCapacity,
		},
	}
}

var parseID = rest.ParseID
