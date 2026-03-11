package modules

import (
	"context"
	"fmt"
)

func init() {
	RegisterModule("copy", ModuleCopy)
}

// ModuleCopy copies files or content to a remote host.
//
// Args:
//
//	src:     source file path (relative to playbook source or absolute)
//	content: direct content string (alternative to src)
//	dest:    destination path on remote (required)
//	mode:    file mode string or integer (optional, default 0644)
func ModuleCopy(ctx context.Context, opts ExecOptions) (string, string, error) {
	dest := stringArg(opts.Args, "dest")
	if dest == "" {
		return "", "", fmt.Errorf("copy: dest is required")
	}

	mode := fileModeArg(opts.Args, "mode", 0644)

	var data []byte
	if content, ok := opts.Args["content"].(string); ok {
		data = []byte(content)
	} else if src, ok := opts.Args["src"].(string); ok && src != "" {
		var err error
		data, err = readSource(opts.Source, src)
		if err != nil {
			return "", "", fmt.Errorf("copy: read source %s: %w", src, err)
		}
	} else {
		return "", "", fmt.Errorf("copy: either src or content is required")
	}

	if err := opts.Connector.PutFile(ctx, data, dest, mode); err != nil {
		return "", "", fmt.Errorf("copy: put file %s: %w", dest, err)
	}

	return fmt.Sprintf("copied to %s", dest), "", nil
}
