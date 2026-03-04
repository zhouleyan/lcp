package apis

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/db"

	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
)

// NewAPIGroupInfos creates all APIGroupInfo instances for the server.
func NewAPIGroupInfos(database *db.DB) []*rest.APIGroupInfo {
	return []*rest.APIGroupInfo{
		iamv1.NewAPIGroupInfo(database),
	}
}
