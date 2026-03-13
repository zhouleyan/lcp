package store

import (
	"context"

	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db/generated"
)

// PGNetworkReader reads networks and subnets directly from PostgreSQL (ACL layer for infra module).
type PGNetworkReader struct {
	queries *generated.Queries
}

// NewPGNetworkReader creates a new PGNetworkReader.
func NewPGNetworkReader(queries *generated.Queries) infra.NetworkReader {
	return &PGNetworkReader{queries: queries}
}

func (r *PGNetworkReader) ListActiveNetworks(ctx context.Context) ([]infra.DBNetworkACLRow, error) {
	return r.queries.ListActiveNetworksWithSubnetCount(ctx)
}

func (r *PGNetworkReader) ListSubnetsByNetworkIDs(ctx context.Context, networkIDs []int64) ([]infra.DBSubnet, error) {
	return r.queries.ListSubnetsByNetworkIDs(ctx, networkIDs)
}
