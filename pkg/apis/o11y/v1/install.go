package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/o11y"
	o11ystore "lcp.io/lcp/pkg/apis/o11y/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of o11y module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewO11yModule initializes the o11y module and builds the API group.
func NewO11yModule(database *db.DB) ModuleResult {
	p := o11y.NewRESTStorageProvider(o11ystore.NewStores(database))

	endpointStorage := o11y.NewEndpointStorage(p.Endpoint)

	group := &rest.APIGroupInfo{
		GroupName: "o11y",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "endpoints",
				Storage: endpointStorage,
			},
		},
	}

	return ModuleResult{Group: group}
}
