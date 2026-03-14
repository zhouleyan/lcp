package setup

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/ansible/connector"
	"lcp.io/lcp/lib/ansible/modules/internal"
	"lcp.io/lcp/lib/ansible/variable"
)

// ModuleSetup gathers facts from the remote host.
// If the connector implements GatherFacts, it calls HostInfo and merges the
// resulting map into the host's RemoteVars via MergeRemoteVariable.
func ModuleSetup(ctx context.Context, opts internal.ExecOptions) (string, string, error) {
	gf, ok := opts.Connector.(connector.GatherFacts)
	if !ok {
		return "", "", fmt.Errorf("setup: connector does not support gather_facts")
	}

	info, err := gf.HostInfo(ctx)
	if err != nil {
		return "", "", fmt.Errorf("setup: %w", err)
	}

	opts.Variable.Merge(variable.MergeRemoteVariable(opts.Host, info))
	return "gathered facts", "", nil
}
