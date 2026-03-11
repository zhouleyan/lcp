package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"lcp.io/lcp/lib/ansible"
	"lcp.io/lcp/lib/ansible/connector"
	"lcp.io/lcp/lib/ansible/converter"
	"lcp.io/lcp/lib/ansible/modules"
	"lcp.io/lcp/lib/ansible/project"
	"lcp.io/lcp/lib/ansible/variable"
)

// PlaybookExecutor orchestrates playbook execution.
type PlaybookExecutor struct {
	inventory  ansible.Inventory
	source     project.Source
	logOutput  io.Writer
	tags       []string
	skipTags   []string
	connectors map[string]connector.Connector
	variable   variable.Variable
}

// Option configures the executor.
type Option func(*PlaybookExecutor)

// WithLogOutput sets the log output writer.
func WithLogOutput(w io.Writer) Option { return func(e *PlaybookExecutor) { e.logOutput = w } }

// WithTags sets the playbook-level --tags filter.
func WithTags(tags []string) Option { return func(e *PlaybookExecutor) { e.tags = tags } }

// WithSkipTags sets the playbook-level --skip-tags filter.
func WithSkipTags(tags []string) Option { return func(e *PlaybookExecutor) { e.skipTags = tags } }

// NewPlaybookExecutor creates a new executor.
func NewPlaybookExecutor(inv ansible.Inventory, source project.Source, opts ...Option) *PlaybookExecutor {
	e := &PlaybookExecutor{
		inventory:  inv,
		source:     source,
		logOutput:  os.Stdout,
		connectors: make(map[string]connector.Connector),
	}
	for _, opt := range opts {
		opt(e)
	}
	e.variable = variable.New(inv)
	return e
}

// Execute runs a playbook and returns the result.
func (e *PlaybookExecutor) Execute(ctx context.Context, playbook *ansible.Playbook) (*ansible.PlaybookResult, error) {
	result := &ansible.PlaybookResult{
		StartTime: time.Now(),
		Stats:     ansible.NewPlaybookStats(),
		Success:   true,
	}

	// Process each play
	for i, play := range playbook.Play {
		if e.logOutput != nil {
			fmt.Fprintf(e.logOutput, "\nPLAY [%s] ***\n", play.Name)
		}

		if err := e.execPlay(ctx, play, result); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("play %d '%s': %v", i+1, play.Name, err)
			result.EndTime = time.Now()
			return result, fmt.Errorf("play %d '%s': %w", i+1, play.Name, err)
		}
	}

	result.EndTime = time.Now()
	return result, nil
}

// execPlay executes a single play.
func (e *PlaybookExecutor) execPlay(ctx context.Context, play ansible.Play, result *ansible.PlaybookResult) error {
	// 1. Resolve hosts from play.PlayHost
	hosts := e.resolveHosts(play.PlayHost)
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts matched for play")
	}

	// 2. Initialize connectors for all hosts
	if err := e.initConnectors(ctx, hosts); err != nil {
		return err
	}
	defer e.closeConnectors(ctx, hosts)

	// 3. Merge play-level variables
	if len(play.Vars.Nodes) > 0 {
		e.variable.Merge(variable.MergeRuntimeVariable(play.Vars.Nodes, hosts...))
	}

	// 4. Load vars_files
	for _, vf := range play.VarsFiles {
		e.loadVarsFile(vf, hosts)
	}

	// 5. Gather facts if enabled
	if play.GatherFacts {
		if err := e.gatherFacts(ctx, hosts); err != nil {
			return fmt.Errorf("gather_facts: %w", err)
		}
	}

	// 6. Handle serial batching
	serial, err := converter.GroupHostBySerial(hosts, play.Serial.Data)
	if err != nil {
		return fmt.Errorf("serial batching: %w", err)
	}

	for _, batch := range serial {
		if err := e.execBatch(ctx, play, batch, result); err != nil {
			return err
		}
	}

	return nil
}

// execBatch executes pre_tasks -> roles -> tasks -> post_tasks for a host batch.
func (e *PlaybookExecutor) execBatch(ctx context.Context, play ansible.Play, hosts []string, result *ansible.PlaybookResult) error {
	taskExec := NewTaskExecutor(e.variable, e.source, e.connectors, e.logOutput)

	makeBlockExec := func() *BlockExecutor {
		be := NewBlockExecutor(taskExec, e.variable, e.source, e.connectors, hosts, e.logOutput)
		be.WithPlayTags(e.tags)
		be.WithPlaySkipTags(e.skipTags)
		be.WithIgnoreErrors(play.IgnoreErrors)
		return be
	}

	// Pre-tasks
	if len(play.PreTasks) > 0 {
		if e.logOutput != nil {
			fmt.Fprintln(e.logOutput, "\nPRE TASKS ***")
		}
		if err := makeBlockExec().Exec(ctx, play.PreTasks); err != nil {
			return fmt.Errorf("pre_tasks: %w", err)
		}
	}

	// Roles
	for _, role := range play.Roles {
		if e.logOutput != nil {
			fmt.Fprintf(e.logOutput, "\nROLE [%s] ***\n", role.Role)
		}
		roleExec := NewRoleExecutor(makeBlockExec(), e.source, e.variable, hosts, e.logOutput)
		if err := roleExec.Exec(ctx, role); err != nil {
			return fmt.Errorf("role '%s': %w", role.Role, err)
		}
	}

	// Tasks
	if len(play.Tasks) > 0 {
		if e.logOutput != nil {
			fmt.Fprintln(e.logOutput, "\nTASKS ***")
		}
		if err := makeBlockExec().Exec(ctx, play.Tasks); err != nil {
			return fmt.Errorf("tasks: %w", err)
		}
	}

	// Post-tasks
	if len(play.PostTasks) > 0 {
		if e.logOutput != nil {
			fmt.Fprintln(e.logOutput, "\nPOST TASKS ***")
		}
		if err := makeBlockExec().Exec(ctx, play.PostTasks); err != nil {
			return fmt.Errorf("post_tasks: %w", err)
		}
	}

	return nil
}

// resolveHosts resolves host names from PlayHost using inventory.
func (e *PlaybookExecutor) resolveHosts(ph ansible.PlayHost) []string {
	result := e.variable.Get(variable.GetHostnames(ph.Hosts))
	if hosts, ok := result.([]string); ok {
		return hosts
	}
	return nil
}

// initConnectors creates connectors for hosts.
func (e *PlaybookExecutor) initConnectors(ctx context.Context, hosts []string) error {
	for _, host := range hosts {
		if _, exists := e.connectors[host]; exists {
			continue
		}
		// Get host variables for connection config
		rawVars := e.variable.Get(variable.GetAllVariable(host))
		vars, ok := rawVars.(map[string]any)
		if !ok {
			vars = make(map[string]any)
		}
		conn, err := connector.NewConnector(host, vars)
		if err != nil {
			return fmt.Errorf("create connector for %s: %w", host, err)
		}
		if err := conn.Init(ctx); err != nil {
			return fmt.Errorf("init connector for %s: %w", host, err)
		}
		e.connectors[host] = conn
	}
	return nil
}

// closeConnectors closes connectors for hosts.
func (e *PlaybookExecutor) closeConnectors(ctx context.Context, hosts []string) {
	for _, host := range hosts {
		if conn, exists := e.connectors[host]; exists {
			_ = conn.Close(ctx)
			delete(e.connectors, host)
		}
	}
}

// gatherFacts runs setup module on all hosts.
func (e *PlaybookExecutor) gatherFacts(ctx context.Context, hosts []string) error {
	setupFn := modules.FindModule("setup")
	if setupFn == nil {
		return fmt.Errorf("setup module not registered")
	}
	for _, host := range hosts {
		_, _, err := setupFn(ctx, modules.ExecOptions{
			Host:      host,
			Variable:  e.variable,
			Connector: e.connectors[host],
			LogOutput: e.logOutput,
		})
		if err != nil {
			return fmt.Errorf("host %s: %w", host, err)
		}
	}
	return nil
}

// loadVarsFile loads a YAML variables file and merges it into runtime vars for the given hosts.
// If the file does not exist or fails to parse, it is silently ignored.
func (e *PlaybookExecutor) loadVarsFile(path string, hosts []string) {
	if e.source == nil {
		return
	}
	data, err := e.source.ReadFile(path)
	if err != nil {
		return // file not found or read error — silently skip
	}
	var vars map[string]any
	if err := yaml.Unmarshal(data, &vars); err != nil {
		return // parse error — silently skip
	}
	if len(vars) == 0 {
		return
	}
	for _, host := range hosts {
		e.variable.Merge(variable.MergeHostRuntimeVars(host, vars))
	}
}
