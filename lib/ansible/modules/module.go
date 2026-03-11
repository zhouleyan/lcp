package modules

import (
	"context"
	"io"
	"sort"
	"sync"

	"lcp.io/lcp/lib/ansible/connector"
	"lcp.io/lcp/lib/ansible/variable"
)

// Source provides access to playbook files (avoids import cycle with project package).
type Source interface {
	ReadFile(path string) ([]byte, error)
}

// ExecOptions contains options for module execution.
type ExecOptions struct {
	Args      map[string]any      // Module arguments
	Host      string              // Target host
	Variable  variable.Variable   // Variable system
	Connector connector.Connector // Host connector
	Source    Source               // Playbook file source
	LogOutput io.Writer           // Log output writer
	WorkDir   string              // Working directory
}

// ModuleExecFunc is the function signature for module execution.
type ModuleExecFunc func(ctx context.Context, opts ExecOptions) (stdout, stderr string, err error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]ModuleExecFunc)
)

// RegisterModule registers a module by name.
func RegisterModule(name string, fn ModuleExecFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = fn
}

// FindModule returns the module function by name, or nil if not found.
func FindModule(name string) ModuleExecFunc {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// IsModule checks if a name is a registered module.
func IsModule(name string) bool {
	return FindModule(name) != nil
}

// ListModules returns all registered module names sorted.
func ListModules() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetAllVariables is a helper that retrieves all variables for the host in ExecOptions.
func (o *ExecOptions) GetAllVariables() map[string]any {
	result := o.Variable.Get(variable.GetAllVariable(o.Host))
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return make(map[string]any)
}
