package apis

import (
	"context"

	"lcp.io/lcp/lib/rest"
	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
	"lcp.io/lcp/pkg/db"
)

// NewAPIGroupInfos assembles all API modules and returns the aggregated APIGroupInfo list.
func NewAPIGroupInfos(ctx context.Context, database *db.DB) []*rest.APIGroupInfo {
	// --- IAM module ---
	iamResult := iamv1.NewIAMModule(ctx, database)

	// --- future modules registered here ---
	// appResult := appv1.NewAppModule(ctx, database)

	return []*rest.APIGroupInfo{
		iamResult.Group,
	}
}
