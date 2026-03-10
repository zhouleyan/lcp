package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/dashboard"
	dashboardstore "lcp.io/lcp/pkg/apis/dashboard/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of Dashboard module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewDashboardModule initializes the Dashboard module.
func NewDashboardModule(database *db.DB) ModuleResult {
	store := dashboardstore.NewPGOverviewStore(database.Queries)

	platformOverview := dashboard.NewPlatformOverviewStorage(store)
	workspaceOverview := dashboard.NewWorkspaceOverviewStorage(store)
	namespaceOverview := dashboard.NewNamespaceOverviewStorage(store)

	group := &rest.APIGroupInfo{
		GroupName: "dashboard",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "overview",
				Storage: platformOverview,
			},
			{
				Name: "workspaces",
				SubResources: []rest.ResourceInfo{
					{
						Name:    "overview",
						Storage: workspaceOverview,
					},
					{
						Name: "namespaces",
						SubResources: []rest.ResourceInfo{
							{
								Name:    "overview",
								Storage: namespaceOverview,
							},
						},
					},
				},
			},
		},
	}

	return ModuleResult{Group: group}
}
