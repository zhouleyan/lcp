package template

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/ansible/modules/internal"
	tmpl "lcp.io/lcp/lib/ansible/template"
)

// ModuleTemplate renders a template file and uploads to remote host.
// Args:
//
//	src: template file path (required)
//	dest: destination path on remote (required)
//	mode: file mode (optional, default 0644)
func ModuleTemplate(ctx context.Context, opts internal.ExecOptions) (string, string, error) {
	src := internal.StringArg(opts.Args, "src")
	dest := internal.StringArg(opts.Args, "dest")
	if src == "" || dest == "" {
		return "", "", fmt.Errorf("template: src and dest are required")
	}

	mode := internal.FileModeArg(opts.Args, "mode", 0644)

	// 1. Read template file from Source.
	data, err := internal.ReadSource(opts.Source, src)
	if err != nil {
		return "", "", fmt.Errorf("template: read %s: %w", src, err)
	}

	// 2. Get all host variables.
	vars := opts.GetAllVariables()

	// 3. Render template.
	rendered, err := tmpl.Parse(vars, string(data))
	if err != nil {
		return "", "", fmt.Errorf("template: render %s: %w", src, err)
	}

	// 4. Upload rendered content.
	if err := opts.Connector.PutFile(ctx, rendered, dest, mode); err != nil {
		return "", "", fmt.Errorf("template: upload %s: %w", dest, err)
	}

	return fmt.Sprintf("templated %s -> %s", src, dest), "", nil
}
