package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/infra"
	infrastore "lcp.io/lcp/pkg/apis/infra/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of Infra module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewInfraModule initializes the Infra module and builds the API group.
func NewInfraModule(database *db.DB) ModuleResult {
	p := infra.NewRESTStorageProvider(infrastore.NewStores(database))

	// Platform-level storages
	platformHostStorage := infra.NewHostStorage(p.Host)
	platformEnvStorage := infra.NewEnvironmentStorage(p.Environment)

	// Workspace-level storages
	wsHostStorage := infra.NewWorkspaceHostStorage(p.Host)
	wsEnvStorage := infra.NewWorkspaceEnvironmentStorage(p.Environment)

	// Namespace-level storages
	nsHostStorage := infra.NewNamespaceHostStorage(p.Host)
	nsEnvStorage := infra.NewNamespaceEnvironmentStorage(p.Environment)

	// CMDB platform-level storages
	regionStorage := infra.NewRegionStorage(p.Region)
	siteStorage := infra.NewSiteStorage(p.Site)
	locationStorage := infra.NewLocationStorage(p.Location)
	rackStorage := infra.NewRackStorage(p.Rack)
	regionSiteStorage := infra.NewRegionSiteStorage(p.Site)
	siteLocationStorage := infra.NewSiteLocationStorage(p.Location)
	locationRackStorage := infra.NewLocationRackStorage(p.Rack)

	// Action handlers
	assignHandler := infra.NewAssignHandler(p.Host, p.HostAssignment)
	unassignHandler := infra.NewUnassignHandler(p.HostAssignment)
	bindEnvHandler := infra.NewBindEnvironmentHandler(p.Host)
	unbindEnvHandler := infra.NewUnbindEnvironmentHandler(p.Host)

	// Custom verb handlers
	hostAssignmentsVerb := infra.NewHostAssignmentsVerb(p.HostAssignment)
	envHostsVerb := infra.NewEnvHostsVerb(p.Environment)

	group := &rest.APIGroupInfo{
		GroupName: "infra",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "hosts",
				Storage: platformHostStorage,
				Actions: []rest.ActionInfo{
					{Name: "assign", Method: "POST", Handler: assignHandler},
					{Name: "unassign", Method: "POST", Handler: unassignHandler},
					{Name: "bind-environment", Method: "POST", Handler: bindEnvHandler},
					{Name: "unbind-environment", Method: "POST", Handler: unbindEnvHandler},
				},
				CustomVerbs: []rest.CustomVerbInfo{
					{Name: "assignments", Storage: hostAssignmentsVerb},
				},
			},
			{
				Name:    "environments",
				Storage: platformEnvStorage,
				CustomVerbs: []rest.CustomVerbInfo{
					{Name: "hosts", Storage: envHostsVerb},
				},
			},
			{
				Name: "workspaces",
				SubResources: []rest.ResourceInfo{
					{
						Name:    "hosts",
						Storage: wsHostStorage,
						Actions: []rest.ActionInfo{
							{Name: "assign", Method: "POST", Handler: assignHandler},
							{Name: "unassign", Method: "POST", Handler: unassignHandler},
							{Name: "bind-environment", Method: "POST", Handler: bindEnvHandler},
							{Name: "unbind-environment", Method: "POST", Handler: unbindEnvHandler},
						},
					},
					{
						Name:    "environments",
						Storage: wsEnvStorage,
						CustomVerbs: []rest.CustomVerbInfo{
							{Name: "hosts", Storage: envHostsVerb},
						},
					},
					{
						Name: "namespaces",
						SubResources: []rest.ResourceInfo{
							{
								Name:    "hosts",
								Storage: nsHostStorage,
								Actions: []rest.ActionInfo{
									{Name: "bind-environment", Method: "POST", Handler: bindEnvHandler},
									{Name: "unbind-environment", Method: "POST", Handler: unbindEnvHandler},
								},
							},
							{
								Name:    "environments",
								Storage: nsEnvStorage,
								CustomVerbs: []rest.CustomVerbInfo{
									{Name: "hosts", Storage: envHostsVerb},
								},
							},
						},
					},
				},
			},
			{
				Name:    "regions",
				Storage: regionStorage,
				SubResources: []rest.ResourceInfo{
					{Name: "sites", Storage: regionSiteStorage},
				},
			},
			{
				Name:    "sites",
				Storage: siteStorage,
				SubResources: []rest.ResourceInfo{
					{Name: "locations", Storage: siteLocationStorage},
				},
			},
			{
				Name:    "locations",
				Storage: locationStorage,
				SubResources: []rest.ResourceInfo{
					{Name: "racks", Storage: locationRackStorage},
				},
			},
			{
				Name:    "racks",
				Storage: rackStorage,
			},
		},
	}

	return ModuleResult{Group: group}
}
