package executor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"lcp.io/lcp/lib/ansible"
	"lcp.io/lcp/lib/ansible/connector"
	"lcp.io/lcp/lib/ansible/modules"
	"lcp.io/lcp/lib/ansible/project"
	"lcp.io/lcp/lib/ansible/template"
	"lcp.io/lcp/lib/ansible/variable"
)

// TaskExecutor executes a single TaskSpec against hosts.
type TaskExecutor struct {
	variable   variable.Variable
	source     project.Source
	logOutput  io.Writer
	connectors map[string]connector.Connector // host -> connector
}

// NewTaskExecutor creates a new task executor.
func NewTaskExecutor(v variable.Variable, src project.Source, conns map[string]connector.Connector, logOutput io.Writer) *TaskExecutor {
	return &TaskExecutor{
		variable:   v,
		source:     src,
		logOutput:  logOutput,
		connectors: conns,
	}
}

// Exec executes a task and returns results per host.
func (e *TaskExecutor) Exec(ctx context.Context, task ansible.TaskSpec) []ansible.TaskResult {
	// 1. Find module
	moduleFn := modules.FindModule(task.Module.Name)
	if moduleFn == nil {
		// Return error result for all hosts
		results := make([]ansible.TaskResult, len(task.Hosts))
		for i, host := range task.Hosts {
			results[i] = ansible.TaskResult{
				Host:   host,
				Status: ansible.TaskStatusFailed,
				Error:  fmt.Sprintf("module %q not found", task.Module.Name),
			}
		}
		return results
	}

	// 2. Resolve loop items
	items := e.resolveLoop(task)

	// 3. Execute on each host in parallel
	var mu sync.Mutex
	results := make([]ansible.TaskResult, 0, len(task.Hosts))
	var wg sync.WaitGroup

	for _, host := range task.Hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := e.execTaskHost(ctx, task, host, moduleFn, items)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(host)
	}
	wg.Wait()

	// 4. Register results if task.Register is set
	if task.Register != "" {
		e.registerResults(task, results)
	}

	return results
}

// execTaskHost executes task on a single host with loop/retry support.
func (e *TaskExecutor) execTaskHost(ctx context.Context, task ansible.TaskSpec, host string, moduleFn modules.ModuleExecFunc, items []any) ansible.TaskResult {
	result := ansible.TaskResult{Host: host, Status: ansible.TaskStatusOK}

	allSkipped := true
	for _, item := range items {
		loopResult := e.executeWithRetry(ctx, task, host, moduleFn, item)

		result.Output = mergeOutput(result.Output, loopResult)

		if loopResult.Result.Status != ansible.TaskStatusSkipped {
			allSkipped = false
		}

		if loopResult.Result.Error != "" {
			if task.IgnoreErrors != nil && *task.IgnoreErrors {
				// Continue loop execution but mark as not fully ok
				allSkipped = false
			} else {
				result.Status = ansible.TaskStatusFailed
				result.Error = loopResult.Result.Error
				return result // stop loop on first failure
			}
		}
	}

	if allSkipped {
		result.Status = ansible.TaskStatusSkipped
	}

	return result
}

// executeWithRetry executes module with retry logic.
func (e *TaskExecutor) executeWithRetry(ctx context.Context, task ansible.TaskSpec, host string, moduleFn modules.ModuleExecFunc, item any) ansible.LoopResult {
	retries := task.Retries
	if retries == 0 {
		retries = 1 // at least one attempt
	}
	delay := task.Delay

	var loopResult ansible.LoopResult

	for attempt := 0; attempt < retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				loopResult.Result.Error = ctx.Err().Error()
				return loopResult
			case <-time.After(time.Duration(delay) * time.Second):
			}
		}

		loopResult = e.executeModule(ctx, task, host, moduleFn, item)

		// Check until conditions
		if len(task.Until) > 0 {
			vars := e.getVarsWithResult(host, loopResult)
			ok, _ := template.ParseBool(vars, task.Until...)
			if ok {
				loopResult.Result.Error = "" // until satisfied, clear error
				break
			}
			// If until not satisfied and this is the last attempt, mark as failure
			if attempt == retries-1 && loopResult.Result.Error == "" {
				loopResult.Result.Error = "until condition not met after retries"
			}
		} else if loopResult.Result.Error == "" {
			break // success, no need to retry
		}
	}

	return loopResult
}

// executeModule executes the module once, handling when/failed_when.
func (e *TaskExecutor) executeModule(ctx context.Context, task ansible.TaskSpec, host string, moduleFn modules.ModuleExecFunc, item any) ansible.LoopResult {
	result := ansible.LoopResult{
		Item: item,
		Result: ansible.TaskResult{
			Host:   host,
			Status: ansible.TaskStatusOK,
		},
	}

	// Get host variables
	rawVars := e.variable.Get(variable.GetAllVariable(host))
	vars, ok := rawVars.(map[string]any)
	if !ok {
		result.Result.Error = fmt.Sprintf("host %s: variables are not a map", host)
		result.Result.Status = ansible.TaskStatusFailed
		return result
	}

	// Set loop item variable
	if item != nil {
		vars["item"] = item
		// Also merge into runtime vars temporarily
		e.variable.Merge(variable.MergeHostRuntimeVars(host, map[string]any{"item": item}))
		defer func() {
			// Clean up loop item variable after execution
			e.variable.Merge(variable.MergeHostRuntimeVars(host, map[string]any{"item": nil}))
		}()
	}

	// Check when conditions
	if len(task.When) > 0 {
		ok, err := template.ParseBool(vars, task.When...)
		if err != nil {
			result.Result.Error = fmt.Sprintf("evaluate when: %v", err)
			result.Result.Status = ansible.TaskStatusFailed
			return result
		}
		if !ok {
			result.Result.Status = ansible.TaskStatusSkipped
			return result // skipped, not an error
		}
	}

	// Render module args with template
	args := e.toArgsMap(task.Module.Args)
	args = e.renderArgs(args, vars)

	// Execute module
	conn := e.connectors[host]
	stdout, stderr, err := moduleFn(ctx, modules.ExecOptions{
		Args:      args,
		Host:      host,
		Variable:  e.variable,
		Connector: conn,
		Source:    e.source,
		LogOutput: e.logOutput,
	})

	result.Result.Output = map[string]any{
		"stdout": strings.TrimRight(stdout, "\n"),
		"stderr": strings.TrimRight(stderr, "\n"),
	}
	if err != nil {
		result.Result.Error = err.Error()
		result.Result.Status = ansible.TaskStatusFailed
	}

	// Check failed_when
	if len(task.FailedWhen) > 0 && result.Result.Error == "" {
		failVars := e.getVarsWithResult(host, result)
		ok, _ := template.ParseBool(failVars, task.FailedWhen...)
		if ok {
			result.Result.Error = "failed_when condition met"
			result.Result.Status = ansible.TaskStatusFailed
		}
	}

	return result
}

// resolveLoop resolves loop items from task.
func (e *TaskExecutor) resolveLoop(task ansible.TaskSpec) []any {
	if task.Loop == nil {
		return []any{nil} // single execution, no loop
	}

	switch v := task.Loop.(type) {
	case []any:
		return v
	case []string:
		items := make([]any, len(v))
		for i, s := range v {
			items[i] = s
		}
		return items
	default:
		// If it's some other type, wrap it as a single-item loop
		return []any{v}
	}
}

// renderArgs renders template syntax in module args.
func (e *TaskExecutor) renderArgs(args map[string]any, vars map[string]any) map[string]any {
	rendered := make(map[string]any, len(args))
	for k, v := range args {
		switch val := v.(type) {
		case string:
			if result, err := template.ParseString(vars, val); err == nil {
				rendered[k] = result
			} else {
				rendered[k] = val
			}
		case map[string]any:
			rendered[k] = e.renderArgs(val, vars)
		default:
			rendered[k] = v
		}
	}
	return rendered
}

// toArgsMap converts the module Args (typed as any) to map[string]any.
func (e *TaskExecutor) toArgsMap(args any) map[string]any {
	if args == nil {
		return make(map[string]any)
	}
	if m, ok := args.(map[string]any); ok {
		return m
	}
	// Fallback: wrap as a single value under "raw"
	return map[string]any{"raw": args}
}

// registerResults registers task results as variables.
func (e *TaskExecutor) registerResults(task ansible.TaskSpec, results []ansible.TaskResult) {
	for _, result := range results {
		regVar := make(map[string]any)

		// Collect loop-level outputs from result.Output
		stdout := extractString(result.Output, "stdout")
		stderr := extractString(result.Output, "stderr")

		regVar["stdout"] = stdout
		regVar["stderr"] = stderr
		regVar["failed"] = result.Error != ""
		regVar["skipped"] = result.Status == ansible.TaskStatusSkipped

		e.variable.Merge(variable.MergeHostRuntimeVars(result.Host, map[string]any{task.Register: regVar}))
	}
}

// getVarsWithResult returns host variables merged with the current loop result,
// so that until/failed_when conditions can reference stdout/stderr/etc.
func (e *TaskExecutor) getVarsWithResult(host string, lr ansible.LoopResult) map[string]any {
	rawVars := e.variable.Get(variable.GetAllVariable(host))
	vars, ok := rawVars.(map[string]any)
	if !ok {
		vars = make(map[string]any)
	}

	// Add result fields so conditions can reference them
	stdout := extractString(lr.Result.Output, "stdout")
	stderr := extractString(lr.Result.Output, "stderr")
	vars["stdout"] = stdout
	vars["stderr"] = stderr
	vars["failed"] = lr.Result.Error != ""

	return vars
}

// mergeOutput merges loop result output into the overall result output map.
func mergeOutput(existing map[string]any, lr ansible.LoopResult) map[string]any {
	if existing == nil {
		return lr.Result.Output
	}
	// For multiple loop iterations, keep the last output
	if lr.Result.Output != nil {
		return lr.Result.Output
	}
	return existing
}

// extractString extracts a string value from a map, returning "" if not found.
func extractString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}
