package set_fact

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/ansible/modules/internal"
	"lcp.io/lcp/lib/ansible/variable"
)

// ModuleSetFact sets runtime variables for the current host.
// Args contains key-value pairs that are merged into the host's RuntimeVars.
func ModuleSetFact(_ context.Context, opts internal.ExecOptions) (string, string, error) {
	if len(opts.Args) == 0 {
		return "", "", fmt.Errorf("set_fact: no facts provided")
	}

	opts.Variable.Merge(variable.MergeHostRuntimeVars(opts.Host, opts.Args))
	return "facts set", "", nil
}
