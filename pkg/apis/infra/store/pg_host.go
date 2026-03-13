package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/ipam"
	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgHostStore struct {
	pool     *pgxpool.Pool
	queries  *generated.Queries
	ipBinder infra.IPBinder
}

// NewPGHostStore creates a new PostgreSQL-backed HostStore.
func NewPGHostStore(pool *pgxpool.Pool, queries *generated.Queries, ipBinder infra.IPBinder) infra.HostStore {
	return &pgHostStore{pool: pool, queries: queries, ipBinder: ipBinder}
}

func (s *pgHostStore) Create(ctx context.Context, host *infra.DBHost, ipConfigs []infra.DBIPConfig) (*infra.DBHost, error) {
	params := generated.CreateHostParams{
		Name:        host.Name,
		DisplayName: host.DisplayName,
		Description: host.Description,
		Hostname:    host.Hostname,
		IpAddress:   host.IpAddress,
		Os:          host.Os,
		Arch:        host.Arch,
		CpuCores:    host.CpuCores,
		MemoryMb:    host.MemoryMb,
		DiskGb:      host.DiskGb,
		Labels:      host.Labels,
		Scope:       host.Scope,
		WorkspaceID: host.WorkspaceID,
		NamespaceID: host.NamespaceID,
		Status:      host.Status,
	}

	// No IP configs — simple INSERT without transaction.
	if len(ipConfigs) == 0 {
		row, err := s.queries.CreateHost(ctx, params)
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return nil, apierrors.NewConflict("host", host.Name)
			}
			return nil, fmt.Errorf("create host: %w", err)
		}
		return &row, nil
	}

	// With IP configs — transactional: INSERT host + allocate IPs.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	row, err := s.queries.WithTx(tx).CreateHost(ctx, params)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("host", host.Name)
		}
		return nil, fmt.Errorf("create host: %w", err)
	}

	for _, cfg := range ipConfigs {
		if err := s.allocateIP(ctx, tx, row.ID, cfg.SubnetID, cfg.IP); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &row, nil
}

// allocateIP locks a subnet, allocates an IP (manual or auto), updates the bitmap,
// and creates an ip_allocation record — all within the provided transaction.
func (s *pgHostStore) allocateIP(ctx context.Context, tx pgx.Tx, hostID, subnetID int64, requestedIP string) error {
	subnet, err := s.ipBinder.GetSubnetForUpdate(ctx, tx, subnetID)
	if err != nil {
		return err
	}

	_, cidr, err := net.ParseCIDR(subnet.Cidr)
	if err != nil {
		return fmt.Errorf("parse subnet CIDR %s: %w", subnet.Cidr, err)
	}

	r, err := ipam.NewCIDRRange(cidr)
	if err != nil {
		return fmt.Errorf("create CIDR range: %w", err)
	}

	if len(subnet.Bitmap) > 0 {
		if err := r.LoadFromBytes(subnet.Bitmap); err != nil {
			return fmt.Errorf("load bitmap: %w", err)
		}
	}

	var allocatedIP net.IP
	if requestedIP != "" {
		ip := net.ParseIP(requestedIP)
		if ip == nil {
			return apierrors.NewBadRequest(fmt.Sprintf("invalid IP address: %s", requestedIP), nil)
		}
		if err := r.Allocate(ip); err != nil {
			if errors.Is(err, ipam.ErrAllocated) {
				return apierrors.NewConflict("ip_allocation", requestedIP)
			}
			if errors.Is(err, ipam.ErrNotInRange) {
				return apierrors.NewBadRequest(fmt.Sprintf("IP %s not in subnet CIDR range %s", requestedIP, subnet.Cidr), nil)
			}
			return fmt.Errorf("allocate IP: %w", err)
		}
		allocatedIP = ip
	} else {
		ip, err := r.AllocateNext()
		if err != nil {
			if errors.Is(err, ipam.ErrFull) {
				return apierrors.NewConflict("subnet", subnet.Cidr)
			}
			return fmt.Errorf("allocate next IP: %w", err)
		}
		allocatedIP = ip
	}

	if err := s.ipBinder.UpdateSubnetBitmap(ctx, tx, subnetID, r.SaveToBytes()); err != nil {
		return err
	}

	_, err = s.ipBinder.CreateIPAllocation(ctx, tx, &infra.DBIPAllocationWithHost{
		SubnetID: subnetID,
		Ip:       allocatedIP.String(),
		HostID:   &hostID,
	})
	return err
}

func (s *pgHostStore) GetByID(ctx context.Context, id int64) (*infra.DBHostWithEnv, error) {
	row, err := s.queries.GetHostByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("host", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get host by id: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) Update(ctx context.Context, host *infra.DBHost) (*infra.DBHost, error) {
	row, err := s.queries.UpdateHost(ctx, generated.UpdateHostParams{
		ID:          host.ID,
		Name:        host.Name,
		DisplayName: host.DisplayName,
		Description: host.Description,
		Hostname:    host.Hostname,
		IpAddress:   host.IpAddress,
		Os:          host.Os,
		Arch:        host.Arch,
		CpuCores:    host.CpuCores,
		MemoryMb:    host.MemoryMb,
		DiskGb:      host.DiskGb,
		Labels:      host.Labels,
		Status:      host.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("host", fmt.Sprintf("%d", host.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("host", host.Name)
		}
		return nil, fmt.Errorf("update host: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBHost, error) {
	params := generated.PatchHostParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["hostname"].(string); ok {
		params.Hostname = &v
	}
	if v, ok := fields["ipAddress"].(string); ok {
		params.IpAddress = &v
	}
	if v, ok := fields["os"].(string); ok {
		params.Os = &v
	}
	if v, ok := fields["arch"].(string); ok {
		params.Arch = &v
	}
	if v, ok := fields["cpuCores"].(float64); ok {
		i := int32(v)
		params.CpuCores = &i
	}
	if v, ok := fields["memoryMb"].(float64); ok {
		i := int64(v)
		params.MemoryMb = &i
	}
	if v, ok := fields["diskGb"].(float64); ok {
		i := int64(v)
		params.DiskGb = &i
	}
	if v, ok := fields["labels"].(map[string]string); ok {
		params.Labels = labelsToJSON(v)
	} else if v, ok := fields["labels"].(map[string]any); ok {
		b, _ := json.Marshal(v)
		params.Labels = b
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}

	row, err := s.queries.PatchHost(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("host", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("host", n)
			}
			return nil, apierrors.NewConflict("host", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch host: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteHost(ctx, id); err != nil {
		return fmt.Errorf("delete host: %w", err)
	}
	return nil
}

func (s *pgHostStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.queries.DeleteHostsByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete hosts by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgHostStore) ListPlatform(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBHostPlatformRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsPlatform(ctx, generated.CountHostsPlatformParams{
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count platform hosts: %w", err)
	}

	rows, err := s.queries.ListHostsPlatform(ctx, generated.ListHostsPlatformParams{
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list platform hosts: %w", err)
	}

	return &db.ListResult[infra.DBHostPlatformRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgHostStore) ListByWorkspaceID(ctx context.Context, wsID int64, q db.ListQuery) (*db.ListResult[infra.DBHostWorkspaceRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsByWorkspaceID(ctx, generated.CountHostsByWorkspaceIDParams{
		WorkspaceID:   &wsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count workspace hosts: %w", err)
	}

	rows, err := s.queries.ListHostsByWorkspaceID(ctx, generated.ListHostsByWorkspaceIDParams{
		WorkspaceID:   &wsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspace hosts: %w", err)
	}

	return &db.ListResult[infra.DBHostWorkspaceRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgHostStore) ListByNamespaceID(ctx context.Context, nsID int64, q db.ListQuery) (*db.ListResult[infra.DBHostNamespaceRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsByNamespaceID(ctx, generated.CountHostsByNamespaceIDParams{
		NamespaceID:   &nsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count namespace hosts: %w", err)
	}

	rows, err := s.queries.ListHostsByNamespaceID(ctx, generated.ListHostsByNamespaceIDParams{
		NamespaceID:   &nsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespace hosts: %w", err)
	}

	return &db.ListResult[infra.DBHostNamespaceRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgHostStore) BindEnvironment(ctx context.Context, hostID, envID int64) error {
	n, err := s.queries.BindHostEnvironment(ctx, generated.BindHostEnvironmentParams{
		ID:            hostID,
		EnvironmentID: &envID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return apierrors.NewConflict("host", fmt.Sprintf("%d", hostID))
	}
	return nil
}

func (s *pgHostStore) UnbindEnvironment(ctx context.Context, hostID int64) error {
	return s.queries.UnbindHostEnvironment(ctx, hostID)
}

func (s *pgHostStore) GetWorkspaceIDByNamespaceID(ctx context.Context, nsID int64) (int64, error) {
	return s.queries.GetWorkspaceIDByNamespaceID(ctx, nsID)
}
