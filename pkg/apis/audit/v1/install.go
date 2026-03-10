package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/audit"
	auditstore "lcp.io/lcp/pkg/apis/audit/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of Audit module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewAuditModule initializes the Audit module.
func NewAuditModule(database *db.DB) ModuleResult {
	store := auditstore.NewPGAuditLogStore(database.Queries)
	logStorage := audit.NewAuditLogStorage(store)

	group := &rest.APIGroupInfo{
		GroupName: "audit",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{Name: "logs", Storage: logStorage},
		},
	}

	return ModuleResult{Group: group}
}
