package network

import (
	"context"
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

	created, err := s.networkStore.Create(ctx, &DBNetwork{
		Name:        n.ObjectMeta.Name,
		DisplayName: n.Spec.DisplayName,
		Description: n.Spec.Description,
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

	updated, err := s.networkStore.Update(ctx, &DBNetwork{
		ID:          nid,
		Name:        n.ObjectMeta.Name,
		DisplayName: n.Spec.DisplayName,
		Description: n.Spec.Description,
		Status:      n.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return networkToAPI(updated), nil
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
		return apierrors.NewConflict("network", id)
	}

	return s.networkStore.Delete(ctx, nid)
}

// DeleteCollection 批量删除网络。
// +openapi:summary=批量删除网络
func (s *networkStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{SuccessCount: len(ids)}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		nid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, nid)
	}

	count, err := s.networkStore.DeleteByIDs(ctx, int64IDs)
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

	// 查询已有 CIDR 做重叠检测
	existingCIDRs, err := s.subnetStore.ListCIDRsByNetworkID(ctx, nid)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("list existing cidrs: %w", err))
	}
	cidrStrs := make([]string, len(existingCIDRs))
	for i, c := range existingCIDRs {
		cidrStrs[i] = c.Cidr
	}

	if errs := ValidateSubnetCreate(subnet.ObjectMeta.Name, &subnet.Spec, cidrStrs); errs.HasErrors() {
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

	status := subnet.Spec.Status
	if status == "" {
		status = "active"
	}

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
		Status:      status,
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
		Status:      subnet.Spec.Status,
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
		return apierrors.NewConflict("subnet", id)
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

	if err := s.subnetStore.Delete(ctx, sid); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("commit transaction: %w", err))
	}

	return nil
}

// DeleteCollection 批量删除子网。
// +openapi:summary=批量删除子网
func (s *subnetStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{SuccessCount: len(ids)}, nil
	}

	networkID := options.PathParams["networkId"]
	nid, err := parseID(networkID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid network ID: %s", networkID), nil)
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		sid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid subnet ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, sid)
	}

	count, err := s.subnetStore.DeleteByIDs(ctx, nid, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== allocationStorage =====

// allocationStorage IP 分配资源的 REST 存储实现，嵌套在 subnets 下。
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
		if err == ipam.ErrAllocated {
			return nil, apierrors.NewConflict("ip_allocation", alloc.Spec.IP)
		}
		if err == ipam.ErrNotInRange {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("IP %s is not within subnet CIDR %s", alloc.Spec.IP, subnet.Cidr), nil)
		}
		return nil, apierrors.NewInternalError(fmt.Errorf("allocate ip: %w", err))
	}

	// SaveToBytes → UPDATE bitmap → 写回
	if err := s.subnetStore.UpdateBitmap(ctx, tx, sid, r.SaveToBytes()); err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("update bitmap: %w", err))
	}

	// INSERT ip_allocation → 记录
	created, err := s.allocStore.Create(ctx, tx, &DBIPAllocation{
		SubnetID:    sid,
		Ip:          alloc.Spec.IP,
		Description: alloc.Spec.Description,
		IsGateway:   false,
	})
	if err != nil {
		return nil, err
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
	// 复用 subnet+ip 查询的方式不太方便，用 ID 查
	// 但我们没有 GetByID 方法，需要通过 subnet+ip 查
	// 实际上 allocationId 就是 ip_allocations.id，直接删除前先查
	// 这里用 list + filter 或直接查——为简单起见，我们 list subnetId 下找到该记录
	// 但更好的做法：先 get，再判断 is_gateway

	// 由于 DeleteOptions 只有 PathParams，我们需要用 allocID 来定位
	// 重新设计：allocationId 参数实际使用 IP 地址值（因为 IP 是 subnet 内唯一的）
	// 但按照 REST 框架约定 allocationId 是 ID，我们按 ID 处理
	// 需要反查 IP 再做 bitmap 释放

	// 方案：先查出分配记录的 IP，再走锁定路径释放
	// 由于没有 GetByID，我们先 list 该 subnet 下所有记录找到对应的
	// 但这不够高效。改用直接在 delete 时传 IP 参数更合理。
	// 然而框架已定为通过 path param 删除，所以我们在 store 层补充一个方法。
	// 为简单起见这里直接走 SQL delete + bitmap update。

	// 用 list 查找——在实践中记录数不大，可接受
	// 更好的做法：在 allocationStore 增加 GetByID，但计划没有这个。
	// 先通过 list 过滤
	result, err := s.allocStore.List(ctx, sid, db.ListQuery{
		Pagination: db.Pagination{Page: 1, PageSize: 10000},
	})
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("list allocations: %w", err))
	}

	var target *DBIPAllocation
	for i := range result.Items {
		if result.Items[i].ID == aid {
			target = &result.Items[i]
			break
		}
	}

	if target == nil {
		return apierrors.NewNotFound("ip_allocation", allocID)
	}

	// gateway 分配不可删除
	if target.IsGateway {
		return apierrors.NewBadRequest("cannot delete gateway IP allocation", nil)
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

	if err := s.allocStore.Delete(ctx, target.ID); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("delete allocation: %w", err))
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
			Status:      s.Status,
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
		}
	}

	return result
}

func allocationToAPI(a *DBIPAllocation) IPAllocation {
	return IPAllocation{
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
}

func networkSpecToPatchFields(n *Network) map[string]any {
	fields := make(map[string]any)
	if n.ObjectMeta.Name != "" {
		fields["name"] = n.ObjectMeta.Name
	}
	if n.Spec.DisplayName != "" {
		fields["displayName"] = n.Spec.DisplayName
	}
	if n.Spec.Description != "" {
		fields["description"] = n.Spec.Description
	}
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
	if s.Spec.DisplayName != "" {
		fields["displayName"] = s.Spec.DisplayName
	}
	if s.Spec.Description != "" {
		fields["description"] = s.Spec.Description
	}
	if s.Spec.Status != "" {
		fields["status"] = s.Spec.Status
	}
	return fields
}
