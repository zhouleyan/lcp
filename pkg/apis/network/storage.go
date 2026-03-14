package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/ipam"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

var parseID = rest.ParseID

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

// ===== networkStorage =====

// networkStorage 网络资源的 REST 存储实现。
type networkStorage struct {
	networkStore NetworkStore
}

// NewNetworkStorage 创建网络 REST 存储。
func NewNetworkStorage(networkStore NetworkStore) rest.StandardStorage {
	return &networkStorage{networkStore: networkStore}
}

func (s *networkStorage) NewObject() runtime.Object { return &Network{} }

// Get 获取网络详情。
// +openapi:summary=获取网络详情
func (s *networkStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["networkId"]
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", id), nil)
	}

	n, err := s.networkStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

	return networkWithCountToAPI(n), nil
}

// List 获取网络列表。
// +openapi:summary=获取网络列表
func (s *networkStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.networkStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Network, len(result.Items))
	for i, item := range result.Items {
		items[i] = networkListRowToAPI(&item)
	}

	return &NetworkList{
		TypeMeta:   runtime.TypeMeta{Kind: "NetworkList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建网络。
// +openapi:summary=创建网络
func (s *networkStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	n, ok := obj.(*Network)
	if !ok {
		return nil, fmt.Errorf("expected *Network, got %T", obj)
	}

	if errs := ValidateNetworkCreate(n.ObjectMeta.Name, &n.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return n, nil
	}

	status := n.Spec.Status
	if status == "" {
		status = "active"
	}

	maxSubnets := n.Spec.MaxSubnets
	if maxSubnets == 0 {
		maxSubnets = 10
	}

	isPublic := true
	if n.Spec.IsPublic != nil {
		isPublic = *n.Spec.IsPublic
	}

	created, err := s.networkStore.Create(ctx, &DBNetwork{
		Name:        n.ObjectMeta.Name,
		DisplayName: n.Spec.DisplayName,
		Description: n.Spec.Description,
		Cidr:        n.Spec.CIDR,
		MaxSubnets:  maxSubnets,
		IsPublic:    isPublic,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return networkToAPI(created), nil
}

// Update 全量更新网络。
// +openapi:summary=更新网络信息（全量）
func (s *networkStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	n, ok := obj.(*Network)
	if !ok {
		return nil, fmt.Errorf("expected *Network, got %T", obj)
	}

	if errs := ValidateNetworkUpdate(&n.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return n, nil
	}

	id := options.PathParams["networkId"]
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", id), nil)
	}

	_, err = s.networkStore.Update(ctx, &DBNetwork{
		ID:          nid,
		Name:        n.ObjectMeta.Name,
		DisplayName: n.Spec.DisplayName,
		Description: n.Spec.Description,
		Status:      n.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	// Re-fetch to get full row with cidr, maxSubnets, isPublic, subnetCount
	full, err := s.networkStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

	return networkWithCountToAPI(full), nil
}

// Patch 部分更新网络。
// +openapi:summary=更新网络信息（部分）
func (s *networkStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	n, ok := obj.(*Network)
	if !ok {
		return nil, fmt.Errorf("expected *Network, got %T", obj)
	}

	id := options.PathParams["networkId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", id), nil)
	}

	fields := networkSpecToPatchFields(n)
	patched, err := s.networkStore.Patch(ctx, nid, fields)
	if err != nil {
		return nil, err
	}

	return networkToAPI(patched), nil
}

// Delete 删除网络（有子网时拒绝）。
// +openapi:summary=删除网络
func (s *networkStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["networkId"]
	nid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", id), nil)
	}

	// 删除保护：有子网时返回 409
	count, err := s.networkStore.CountSubnets(ctx, nid)
	if err != nil {
		return err
	}
	if count > 0 {
		return apierrors.NewConflictMessage("cannot delete network: subnets still exist")
	}

	return s.networkStore.Delete(ctx, nid)
}

// DeleteCollection 批量删除网络（逐个检查删除保护）。
// +openapi:summary=批量删除网络
func (s *networkStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{SuccessCount: len(ids)}, nil
	}

	// 逐个检查删除保护，收集可删除的 ID
	var deletableIDs []int64
	for _, id := range ids {
		nid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", id), nil)
		}
		count, err := s.networkStore.CountSubnets(ctx, nid)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, apierrors.NewConflictMessage("cannot delete network: subnets still exist")
		}
		deletableIDs = append(deletableIDs, nid)
	}

	count, err := s.networkStore.DeleteByIDs(ctx, deletableIDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== subnetStorage =====

// subnetStorage 子网资源的 REST 存储实现，嵌套在 networks 下。
// +openapi:path=/networks/{networkId}/subnets
type subnetStorage struct {
	subnetStore  SubnetStore
	allocStore   IPAllocationStore
	networkStore NetworkStore
}

// NewSubnetStorage 创建子网 REST 存储。
func NewSubnetStorage(subnetStore SubnetStore, allocStore IPAllocationStore, networkStore NetworkStore) rest.StandardStorage {
	return &subnetStorage{
		subnetStore:  subnetStore,
		allocStore:   allocStore,
		networkStore: networkStore,
	}
}

func (s *subnetStorage) NewObject() runtime.Object { return &Subnet{} }

// Get 获取子网详情。
// +openapi:summary=获取子网详情
func (s *subnetStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["subnetId"]
	sid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", id), nil)
	}

	subnet, err := s.subnetStore.GetByID(ctx, sid)
	if err != nil {
		return nil, err
	}

	return subnetToAPI(subnet), nil
}

// List 获取子网列表。
// +openapi:summary=获取子网列表
func (s *subnetStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	networkID := options.PathParams["networkId"]
	nid, err := parseID(networkID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", networkID), nil)
	}

	query := restOptionsToListQuery(options)

	// usage 排序需要在 Go 层计算后排序
	if query.SortBy == "usage" {
		return s.listSortedByUsage(ctx, nid, query)
	}

	result, err := s.subnetStore.List(ctx, nid, query)
	if err != nil {
		return nil, err
	}

	items := make([]Subnet, len(result.Items))
	for i, item := range result.Items {
		items[i] = *subnetToAPI(&item)
	}

	return &SubnetList{
		TypeMeta:   runtime.TypeMeta{Kind: "SubnetList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// listSortedByUsage 按 IP 使用率排序（需在 Go 层计算 bitmap 后排序）。
// 注意：加载最多 10000 条子网到内存排序，maxSubnets 上限为 50，实际不会触及此限制。
func (s *subnetStorage) listSortedByUsage(ctx context.Context, nid int64, query db.ListQuery) (runtime.Object, error) {
	allQuery := query
	allQuery.SortBy = ""
	allQuery.Pagination = db.Pagination{Page: 1, PageSize: 10000}

	result, err := s.subnetStore.List(ctx, nid, allQuery)
	if err != nil {
		return nil, err
	}

	items := make([]Subnet, len(result.Items))
	for i, item := range result.Items {
		items[i] = *subnetToAPI(&item)
	}

	sort.Slice(items, func(i, j int) bool {
		pi := subnetUsagePercent(&items[i])
		pj := subnetUsagePercent(&items[j])
		if query.SortOrder == "asc" {
			return pi < pj
		}
		return pi > pj
	})

	total := len(items)
	start := (query.Pagination.Page - 1) * query.Pagination.PageSize
	end := start + query.Pagination.PageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &SubnetList{
		TypeMeta:   runtime.TypeMeta{Kind: "SubnetList"},
		Items:      items[start:end],
		TotalCount: int64(total),
	}, nil
}

func subnetUsagePercent(s *Subnet) float64 {
	if s.Spec.TotalIPs == 0 {
		return 0
	}
	return float64(s.Spec.UsedIPs) / float64(s.Spec.TotalIPs)
}

// Create 创建子网。
// +openapi:summary=创建子网
func (s *subnetStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	subnet, ok := obj.(*Subnet)
	if !ok {
		return nil, fmt.Errorf("expected *Subnet, got %T", obj)
	}

	networkID := options.PathParams["networkId"]
	nid, err := parseID(networkID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", networkID), nil)
	}

	// 查询网络（用于子网范围校验和数量限制）
	networkObj, err := s.networkStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

	// 子网数量限制检查
	if networkObj.SubnetCount >= int64(networkObj.MaxSubnets) {
		return nil, apierrors.NewConflictMessage(fmt.Sprintf("network has reached the maximum number of subnets (%d)", networkObj.MaxSubnets))
	}

	// 查询已有 CIDR 做重叠检测
	existingCIDRs, err := s.subnetStore.ListCIDRsByNetworkID(ctx, nid)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("list existing cidrs: %w", err))
	}
	cidrStrs := make([]string, len(existingCIDRs))
	for i, c := range existingCIDRs {
		cidrStrs[i] = c.Cidr
	}

	if errs := ValidateSubnetCreate(subnet.ObjectMeta.Name, &subnet.Spec, cidrStrs, networkObj.Cidr); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return subnet, nil
	}

	// 解析 CIDR 并创建 Range
	_, cidrNet, _ := net.ParseCIDR(subnet.Spec.CIDR)
	r, err := ipam.NewCIDRRange(cidrNet)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid CIDR range: %v", err), nil)
	}

	// 如有 gateway 则预分配
	var gatewayIP net.IP
	if subnet.Spec.Gateway != "" {
		gatewayIP = net.ParseIP(subnet.Spec.Gateway)
		if err := r.Allocate(gatewayIP); err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("cannot allocate gateway IP: %v", err), nil)
		}
	}

	bitmap := r.SaveToBytes()

	// 事务：创建 subnet + 创建 gateway allocation
	tx, err := s.subnetStore.BeginTx(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := s.subnetStore.Create(ctx, tx, &DBSubnet{
		Name:        subnet.ObjectMeta.Name,
		DisplayName: subnet.Spec.DisplayName,
		Description: subnet.Spec.Description,
		NetworkID:   nid,
		Cidr:        subnet.Spec.CIDR,
		Gateway:     subnet.Spec.Gateway,
		Bitmap:      bitmap,
	})
	if err != nil {
		return nil, err
	}

	// 创建 gateway 的 IP 分配记录
	if gatewayIP != nil {
		_, err = s.allocStore.Create(ctx, tx, &DBIPAllocation{
			SubnetID:    created.ID,
			Ip:          gatewayIP.String(),
			Description: "Gateway",
			IsGateway:   true,
		})
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("create gateway allocation: %w", err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("commit transaction: %w", err))
	}

	return subnetToAPI(created), nil
}

// Update 全量更新子网（不可修改 CIDR 和 gateway）。
// +openapi:summary=更新子网信息（全量）
func (s *subnetStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	subnet, ok := obj.(*Subnet)
	if !ok {
		return nil, fmt.Errorf("expected *Subnet, got %T", obj)
	}

	if errs := ValidateSubnetUpdate(&subnet.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return subnet, nil
	}

	id := options.PathParams["subnetId"]
	sid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", id), nil)
	}

	updated, err := s.subnetStore.Update(ctx, &DBSubnet{
		ID:          sid,
		Name:        subnet.ObjectMeta.Name,
		DisplayName: subnet.Spec.DisplayName,
		Description: subnet.Spec.Description,
	})
	if err != nil {
		return nil, err
	}

	return subnetToAPI(updated), nil
}

// Patch 部分更新子网。
// +openapi:summary=更新子网信息（部分）
func (s *subnetStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	subnet, ok := obj.(*Subnet)
	if !ok {
		return nil, fmt.Errorf("expected *Subnet, got %T", obj)
	}

	id := options.PathParams["subnetId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	sid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", id), nil)
	}

	fields := subnetSpecToPatchFields(subnet)
	patched, err := s.subnetStore.Patch(ctx, sid, fields)
	if err != nil {
		return nil, err
	}

	return subnetToAPI(patched), nil
}

// Delete 删除子网（有非 gateway 分配时拒绝）。
// +openapi:summary=删除子网
func (s *subnetStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["subnetId"]
	sid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", id), nil)
	}

	// 删除保护：有非 gateway 分配时返回 409
	count, err := s.subnetStore.CountNonGatewayAllocations(ctx, sid)
	if err != nil {
		return err
	}
	if count > 0 {
		return apierrors.NewConflictMessage("cannot delete subnet: IP allocations still exist")
	}

	// 事务内先删所有 ip_allocations 再删 subnet
	tx, err := s.subnetStore.BeginTx(ctx)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.allocStore.DeleteBySubnetID(ctx, tx, sid); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("delete allocations: %w", err))
	}

	if err := s.subnetStore.DeleteTx(ctx, tx, sid); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("commit transaction: %w", err))
	}

	return nil
}

// DeleteCollection 批量删除子网（逐个检查删除保护，事务内清理 allocations）。
// +openapi:summary=批量删除子网
func (s *subnetStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{SuccessCount: len(ids)}, nil
	}

	// 逐个检查删除保护
	var int64IDs []int64
	for _, id := range ids {
		sid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", id), nil)
		}
		count, err := s.subnetStore.CountNonGatewayAllocations(ctx, sid)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, apierrors.NewConflictMessage("cannot delete subnet: IP allocations still exist")
		}
		int64IDs = append(int64IDs, sid)
	}

	// 事务内先删 allocations 再删 subnets
	tx, err := s.subnetStore.BeginTx(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, sid := range int64IDs {
		if err := s.allocStore.DeleteBySubnetID(ctx, tx, sid); err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("delete allocations for subnet %d: %w", sid, err))
		}
		if err := s.subnetStore.DeleteTx(ctx, tx, sid); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("commit transaction: %w", err))
	}

	return &rest.DeletionResult{
		SuccessCount: len(int64IDs),
	}, nil
}

// ===== allocationStorage =====

// allocationStorage IP 分配资源的 REST 存储实现，嵌套在 subnets 下。
// +openapi:path=/networks/{networkId}/subnets/{subnetId}/allocations
// +openapi:resource=IPAllocation
type allocationStorage struct {
	allocStore  IPAllocationStore
	subnetStore SubnetStore
}

// NewAllocationStorage 创建 IP 分配 REST 存储。
func NewAllocationStorage(allocStore IPAllocationStore, subnetStore SubnetStore) rest.Storage {
	return &allocationStorage{
		allocStore:  allocStore,
		subnetStore: subnetStore,
	}
}

func (s *allocationStorage) NewObject() runtime.Object { return &IPAllocation{} }

// List 获取 IP 分配列表。
// +openapi:summary=获取 IP 分配列表
func (s *allocationStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	subnetID := options.PathParams["subnetId"]
	sid, err := parseID(subnetID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", subnetID), nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.allocStore.List(ctx, sid, query)
	if err != nil {
		return nil, err
	}

	items := make([]IPAllocation, len(result.Items))
	for i, item := range result.Items {
		items[i] = allocationToAPI(&item)
	}

	return &IPAllocationList{
		TypeMeta:   runtime.TypeMeta{Kind: "IPAllocationList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 分配 IP（bitmap 锁定路径）。
// +openapi:summary=分配 IP 地址
func (s *allocationStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	alloc, ok := obj.(*IPAllocation)
	if !ok {
		return nil, fmt.Errorf("expected *IPAllocation, got %T", obj)
	}

	subnetID := options.PathParams["subnetId"]
	sid, err := parseID(subnetID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", subnetID), nil)
	}

	if errs := ValidateIPAllocationCreate(&alloc.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return alloc, nil
	}

	ip := net.ParseIP(alloc.Spec.IP)

	// BEGIN TX
	tx, err := s.subnetStore.BeginTx(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SELECT subnet FOR UPDATE → 行锁 + 获取 bitmap
	subnet, err := s.subnetStore.GetByIDForUpdate(ctx, tx, sid)
	if err != nil {
		return nil, err
	}

	// NewCIDRRange + LoadFromBytes → 恢复 bitmap
	_, cidrNet, err := net.ParseCIDR(subnet.Cidr)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("parse subnet cidr: %w", err))
	}

	r, err := ipam.NewCIDRRange(cidrNet)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("create cidr range: %w", err))
	}

	if len(subnet.Bitmap) > 0 {
		if err := r.LoadFromBytes(subnet.Bitmap); err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("restore bitmap: %w", err))
		}
	}

	// Allocate(ip) → 标记分配
	if err := r.Allocate(ip); err != nil {
		if errors.Is(err, ipam.ErrAllocated) {
			return nil, apierrors.NewConflict("ip_allocation", alloc.Spec.IP)
		}
		if errors.Is(err, ipam.ErrNotInRange) {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("IP %s is not within subnet CIDR %s", alloc.Spec.IP, subnet.Cidr), nil)
		}
		return nil, apierrors.NewInternalError(fmt.Errorf("allocate ip: %w", err))
	}

	// SaveToBytes → UPDATE bitmap → 写回
	if err := s.subnetStore.UpdateBitmap(ctx, tx, sid, r.SaveToBytes()); err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("update bitmap: %w", err))
	}

	// 如果请求设为网关，检查子网是否已有网关
	isGateway := alloc.Spec.IsGateway
	if isGateway && subnet.Gateway != "" {
		return nil, apierrors.NewConflictMessage(fmt.Sprintf("subnet already has gateway %s", subnet.Gateway))
	}

	// INSERT ip_allocation → 记录
	created, err := s.allocStore.Create(ctx, tx, &DBIPAllocation{
		SubnetID:    sid,
		Ip:          alloc.Spec.IP,
		Description: alloc.Spec.Description,
		IsGateway:   isGateway,
	})
	if err != nil {
		return nil, err
	}

	// 设为网关时更新子网的 gateway 字段
	if isGateway {
		if err := s.subnetStore.UpdateGateway(ctx, tx, sid, alloc.Spec.IP); err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("update subnet gateway: %w", err))
		}
	}

	// COMMIT
	if err := tx.Commit(ctx); err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("commit transaction: %w", err))
	}

	return &IPAllocation{
		TypeMeta: runtime.TypeMeta{Kind: "IPAllocation"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(created.ID, 10),
			CreatedAt: &created.CreatedAt,
		},
		Spec: IPAllocationSpec{
			IP:          created.Ip,
			Description: created.Description,
			IsGateway:   created.IsGateway,
			SubnetID:    strconv.FormatInt(created.SubnetID, 10),
		},
	}, nil
}

// Delete 释放 IP（gateway 不可删除）。
// +openapi:summary=释放 IP 地址
func (s *allocationStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	subnetID := options.PathParams["subnetId"]
	sid, err := parseID(subnetID)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", subnetID), nil)
	}

	allocID := options.PathParams["allocationId"]
	aid, err := parseID(allocID)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid allocation ID: %s", allocID), nil)
	}

	// 查询分配记录
	target, err := s.allocStore.GetByID(ctx, aid)
	if err != nil {
		return err
	}

	// 校验 allocation 属于该 subnet
	if target.SubnetID != sid {
		return apierrors.NewNotFound("ip_allocation", allocID)
	}

	// 已绑定主机的 IP 不允许直接删除，需先从主机解绑
	if target.HostID != nil {
		return apierrors.NewBadRequest("cannot delete IP bound to a host, unbind it first", nil)
	}

	ip := net.ParseIP(target.Ip)

	// 锁定路径释放 bitmap
	tx, err := s.subnetStore.BeginTx(ctx)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("begin transaction: %w", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	subnet, err := s.subnetStore.GetByIDForUpdate(ctx, tx, sid)
	if err != nil {
		return err
	}

	_, cidrNet, err := net.ParseCIDR(subnet.Cidr)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("parse subnet cidr: %w", err))
	}

	r, err := ipam.NewCIDRRange(cidrNet)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("create cidr range: %w", err))
	}

	if len(subnet.Bitmap) > 0 {
		if err := r.LoadFromBytes(subnet.Bitmap); err != nil {
			return apierrors.NewInternalError(fmt.Errorf("restore bitmap: %w", err))
		}
	}

	r.Release(ip)

	if err := s.subnetStore.UpdateBitmap(ctx, tx, sid, r.SaveToBytes()); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("update bitmap: %w", err))
	}

	if err := s.allocStore.DeleteTx(ctx, tx, target.ID); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("delete allocation: %w", err))
	}

	// 释放网关 IP 时清空子网 gateway 字段
	if target.IsGateway {
		if err := s.subnetStore.UpdateGateway(ctx, tx, sid, ""); err != nil {
			return apierrors.NewInternalError(fmt.Errorf("clear subnet gateway: %w", err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("commit transaction: %w", err))
	}

	return nil
}

// ===== 类型转换辅助函数 =====

func networkToAPI(n *DBNetwork) *Network {
	return &Network{
		TypeMeta: runtime.TypeMeta{Kind: "Network"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(n.ID, 10),
			Name:      n.Name,
			CreatedAt: &n.CreatedAt,
			UpdatedAt: &n.UpdatedAt,
		},
		Spec: NetworkSpec{
			DisplayName: n.DisplayName,
			Description: n.Description,
			CIDR:        n.Cidr,
			MaxSubnets:  n.MaxSubnets,
			IsPublic:    &n.IsPublic,
			Status:      n.Status,
		},
	}
}

func networkWithCountToAPI(n *DBNetworkWithCount) *Network {
	return &Network{
		TypeMeta: runtime.TypeMeta{Kind: "Network"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(n.ID, 10),
			Name:      n.Name,
			CreatedAt: &n.CreatedAt,
			UpdatedAt: &n.UpdatedAt,
		},
		Spec: NetworkSpec{
			DisplayName: n.DisplayName,
			Description: n.Description,
			CIDR:        n.Cidr,
			MaxSubnets:  n.MaxSubnets,
			IsPublic:    &n.IsPublic,
			Status:      n.Status,
			SubnetCount: n.SubnetCount,
		},
	}
}

func networkListRowToAPI(n *DBNetworkListRow) Network {
	return Network{
		TypeMeta: runtime.TypeMeta{Kind: "Network"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(n.ID, 10),
			Name:      n.Name,
			CreatedAt: &n.CreatedAt,
			UpdatedAt: &n.UpdatedAt,
		},
		Spec: NetworkSpec{
			DisplayName: n.DisplayName,
			Description: n.Description,
			CIDR:        n.Cidr,
			MaxSubnets:  n.MaxSubnets,
			IsPublic:    &n.IsPublic,
			Status:      n.Status,
			SubnetCount: n.SubnetCount,
		},
	}
}

func subnetToAPI(s *DBSubnet) *Subnet {
	result := &Subnet{
		TypeMeta: runtime.TypeMeta{Kind: "Subnet"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(s.ID, 10),
			Name:      s.Name,
			CreatedAt: &s.CreatedAt,
			UpdatedAt: &s.UpdatedAt,
		},
		Spec: SubnetSpec{
			DisplayName: s.DisplayName,
			Description: s.Description,
			CIDR:        s.Cidr,
			Gateway:     s.Gateway,
			NetworkID:   strconv.FormatInt(s.NetworkID, 10),
		},
	}

	// 从 bitmap 计算 IP 使用统计
	_, cidrNet, err := net.ParseCIDR(s.Cidr)
	if err == nil {
		r, err := ipam.NewCIDRRange(cidrNet)
		if err == nil {
			if len(s.Bitmap) > 0 {
				_ = r.LoadFromBytes(s.Bitmap)
			}
			result.Spec.FreeIPs = r.Free()
			result.Spec.UsedIPs = r.Used()
			result.Spec.TotalIPs = r.Free() + r.Used()
			if nextIP, err := r.NextFree(); err == nil {
				result.Spec.NextFreeIP = nextIP.String()
			}
		}
	}

	return result
}

func allocationToAPI(a *DBIPAllocationListRow) IPAllocation {
	alloc := IPAllocation{
		TypeMeta: runtime.TypeMeta{Kind: "IPAllocation"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(a.ID, 10),
			CreatedAt: &a.CreatedAt,
		},
		Spec: IPAllocationSpec{
			IP:          a.Ip,
			Description: a.Description,
			IsGateway:   a.IsGateway,
			SubnetID:    strconv.FormatInt(a.SubnetID, 10),
		},
	}
	if a.HostID != nil {
		alloc.Spec.HostID = strconv.FormatInt(*a.HostID, 10)
	}
	if a.HostName != nil {
		alloc.Spec.HostName = *a.HostName
	}
	return alloc
}

func networkSpecToPatchFields(n *Network) map[string]any {
	fields := make(map[string]any)
	if n.ObjectMeta.Name != "" {
		fields["name"] = n.ObjectMeta.Name
	}
	// Always include displayName and description so they can be cleared to empty
	fields["displayName"] = n.Spec.DisplayName
	fields["description"] = n.Spec.Description
	if n.Spec.Status != "" {
		fields["status"] = n.Spec.Status
	}
	return fields
}

func subnetSpecToPatchFields(s *Subnet) map[string]any {
	fields := make(map[string]any)
	if s.ObjectMeta.Name != "" {
		fields["name"] = s.ObjectMeta.Name
	}
	// Always include displayName and description so they can be cleared to empty
	fields["displayName"] = s.Spec.DisplayName
	fields["description"] = s.Spec.Description
	return fields
}
