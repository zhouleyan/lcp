package openapi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// TypeInfo holds parsed information about an API type.
type TypeInfo struct {
	Name        string
	Package     string
	Fields      []FieldInfo
	Annotations []Annotation
	Description string
	IsListType  bool
}

// FieldInfo holds parsed information about a struct field.
type FieldInfo struct {
	Name        string
	JSONName    string
	GoType      string
	OmitEmpty   bool
	Annotations FieldAnnotations
}

// GroupInfo holds parsed information about an API group.
type GroupInfo struct {
	GroupName    string
	GroupVersion string
	ModuleName   string
	Types        []TypeInfo
}

// Parser scans Go source files for OpenAPI type definitions.
type Parser struct {
	rootDir string
}

// NewParser creates a parser that will scan from the given root directory.
func NewParser(rootDir string) *Parser {
	return &Parser{rootDir: rootDir}
}

// Parse scans the root directory for API type definitions.
// It looks for Go files with +openapi: annotations in each resource directory
// (e.g., pkg/apis/iam/) where doc.go and types.go define the API group.
func (p *Parser) Parse() ([]GroupInfo, error) {
	var groups []GroupInfo

	entries, err := os.ReadDir(p.rootDir)
	if err != nil {
		return nil, fmt.Errorf("read apis dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse the resource directory itself (e.g., pkg/apis/iam/)
		// which contains doc.go (with group annotations) and types.go (with type annotations)
		resourceDir := filepath.Join(p.rootDir, entry.Name())
		group, err := p.parseGroup(resourceDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parse group %s: %w", entry.Name(), err)
		}
		if group != nil {
			groups = append(groups, *group)
		}
	}

	return groups, nil
}

func (p *Parser) parseGroup(dir string, dirName string) (*GroupInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse directory %s: %w", dir, err)
	}

	group := &GroupInfo{
		ModuleName: dirName,
	}

	for _, pkg := range pkgs {
		// Check for doc.go package annotations
		for _, file := range pkg.Files {
			pa := ParsePackageAnnotations(file)
			if pa.GroupVersion != "" {
				group.GroupName = pa.GroupName
				group.GroupVersion = pa.GroupVersion
			}
			if pa.ModuleName != "" {
				group.ModuleName = pa.ModuleName
			}
		}

		// Collect type definitions
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					annotations := ParseAnnotations(genDecl.Doc)
					if len(annotations) == 0 {
						// Skip types without +openapi: annotations
						continue
					}

					// Extract description from type-level annotations
					var description string
					for _, ann := range annotations {
						if ann.Key == "description" {
							description = ann.Value
							break
						}
					}

					typeInfo := TypeInfo{
						Name:        typeSpec.Name.Name,
						Package:     pkg.Name,
						Annotations: annotations,
						Description: description,
						IsListType:  strings.HasSuffix(typeSpec.Name.Name, "List"),
					}

					for _, field := range structType.Fields.List {
						fi := parseField(field)
						if fi != nil {
							typeInfo.Fields = append(typeInfo.Fields, *fi)
						}
					}

					group.Types = append(group.Types, typeInfo)
				}
			}
		}
	}

	if len(group.Types) == 0 {
		return nil, nil
	}

	return group, nil
}

func parseField(field *ast.Field) *FieldInfo {
	if len(field.Names) == 0 {
		// Embedded field — skip for now
		return nil
	}

	name := field.Names[0].Name
	jsonName := name
	omitEmpty := false

	// Parse json tag
	if field.Tag != nil {
		tag := strings.Trim(field.Tag.Value, "`")
		if jsonTag := extractTag(tag, "json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" {
				if parts[0] != "" {
					jsonName = parts[0]
				}
				for _, p := range parts[1:] {
					if p == "omitempty" {
						omitEmpty = true
					}
				}
			}
		}
	}

	return &FieldInfo{
		Name:        name,
		JSONName:    jsonName,
		GoType:      typeString(field.Type),
		OmitEmpty:   omitEmpty,
		Annotations: ParseFieldAnnotations(field.Doc),
	}
}

func extractTag(tag, key string) string {
	search := key + `:`
	idx := strings.Index(tag, search)
	if idx < 0 {
		return ""
	}
	val := tag[idx+len(search):]
	if len(val) == 0 {
		return ""
	}
	quote := val[0]
	if quote != '"' {
		return ""
	}
	end := strings.IndexByte(val[1:], quote)
	if end < 0 {
		return ""
	}
	return val[1 : end+1]
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	default:
		return "interface{}"
	}
}
