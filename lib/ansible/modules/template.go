package modules

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/ansible/template"
)

func init() {
	RegisterModule("template", ModuleTemplate)
}

// ModuleTemplate renders a template file and uploads to remote host.
// Args:
//
//	src: template file path (required)
//	dest: destination path on remote (required)
//	mode: file mode (optional, default 0644)
func ModuleTemplate(ctx context.Context, opts ExecOptions) (string, string, error) {
	src := stringArg(opts.Args, "src")
	dest := stringArg(opts.Args, "dest")
	if src == "" || dest == "" {
		return "", "", fmt.Errorf("template: src and dest are required")
	}

	mode := fileModeArg(opts.Args, "mode", 0644)

	// 1. Read template file from Source.
	data, err := readSource(opts.Source, src)
	if err != nil {
		return "", "", fmt.Errorf("template: read %s: %w", src, err)
	}

	// 2. Get all host variables.
	vars := opts.GetAllVariables()

	// 3. Render template.
	rendered, err := template.Parse(vars, string(data))
	if err != nil {
		return "", "", fmt.Errorf("template: render %s: %w", src, err)
	}

	// 4. Upload rendered content.
	if err := opts.Connector.PutFile(ctx, rendered, dest, mode); err != nil {
		return "", "", fmt.Errorf("template: upload %s: %w", dest, err)
	}

	return fmt.Sprintf("templated %s -> %s", src, dest), "", nil
}
