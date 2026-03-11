package modules

import (
	"context"
	"fmt"
)

func init() {
	RegisterModule("command", ModuleCommand)
	RegisterModule("shell", ModuleShell)
}

// ModuleCommand executes a command on the remote host via the connector.
// Args: map with "cmd", "command", or "shell" key containing the command string.
func ModuleCommand(ctx context.Context, opts ExecOptions) (string, string, error) {
	cmd := extractCommand(opts.Args)
	if cmd == "" {
		return "", "", fmt.Errorf("command module: no command specified")
	}

	stdout, stderr, err := opts.Connector.ExecuteCommand(ctx, cmd)
	return string(stdout), string(stderr), err
}

// ModuleShell executes a command via shell. The connector already wraps
// commands in a shell, so this is functionally identical to ModuleCommand.
func ModuleShell(ctx context.Context, opts ExecOptions) (string, string, error) {
	return ModuleCommand(ctx, opts)
}

// extractCommand extracts the command string from module args.
// It checks "cmd", "command", and "shell" keys in order.
func extractCommand(args map[string]any) string {
	for _, key := range []string{"cmd", "command", "shell"} {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
