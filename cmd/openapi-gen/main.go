package main

import (
	"flag"
	"fmt"
	"os"

	"lcp.io/lcp/lib/openapi"
)

func main() {
	var (
		apisDir string
		output  string
		format  string
		title   string
		version string
	)

	flag.StringVar(&apisDir, "apis-dir", "pkg/apis", "Directory containing API type definitions")
	flag.StringVar(&output, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&format, "format", "json", "Output format: json or yaml")
	flag.StringVar(&title, "title", "LCP API", "API title")
	flag.StringVar(&version, "version", "v1", "API version")
	flag.Parse()

	parser := openapi.NewParser(apisDir)
	groups, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing API types: %v\n", err)
		os.Exit(1)
	}

	generator := openapi.NewGenerator(title, "LCP Platform API", version)
	doc := generator.Generate(groups)

	var w *os.File
	if output != "" {
		w, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	switch format {
	case "yaml":
		if err := openapi.WriteYAML(w, doc); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing YAML: %v\n", err)
			os.Exit(1)
		}
	default:
		if err := openapi.WriteJSON(w, doc); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
			os.Exit(1)
		}
	}
}
