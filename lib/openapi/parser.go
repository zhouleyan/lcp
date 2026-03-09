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
	Name              string
	Package           string
	Fields            []FieldInfo
	Annotations       []Annotation
	Description       string
	IsListType        bool
	SchemaOnly        bool              // +openapi:schema — register in components/schemas but do not generate CRUD paths
	Paths             []string          // from +openapi:path annotations
	OperationSummary  map[string]string // from +openapi:summary.METHOD= (e.g. "list", "create", "get", "update", "patch", "delete", "deleteCollection")
	ActionSummary     map[string]string // from +openapi:action.NAME.summary= (e.g. "change-password")
	CustomVerbSummary map[string]string // from +openapi:customverb= on standalone functions (e.g. "workspaces" → summary)
	CustomVerbResponse map[string]string // from +openapi:response= on custom verb functions (e.g. "rolebindings" → "RoleBindingList")
}

// FieldInfo holds parsed information about a struct field.
type FieldInfo struct {
	Name        string
	JSONName    string
	GoType      string
	OmitEmpty   bool
	Annotations FieldAnnotations
}

// EndpointInfo holds parsed information about a standalone HTTP endpoint
// defined via +openapi:endpoint annotations on functions.
type EndpointInfo struct {
	Path        string            // +openapi:path=...
	Method      string            // +openapi:method=GET|POST|PUT|PATCH|DELETE
	Summary     string            // +openapi:summary=...
	Description string            // +openapi:description=...
	Tag         string            // +openapi:tag=...
	OperationID string            // +openapi:operationId=...
	RequestBody *EndpointBody     // +openapi:requestBody.contentType=... and +openapi:requestBody.schema=...
	Responses   []EndpointResponse // +openapi:response.CODE.description=...
}

// EndpointBody describes the request body of a standalone endpoint.
type EndpointBody struct {
	ContentType string // e.g. "application/x-www-form-urlencoded", "application/json"
	SchemaRef   string // reference name or empty for generic object
}

// EndpointResponse describes a response of a standalone endpoint.
type EndpointResponse struct {
	StatusCode  string
	Description string
	ContentType string
	SchemaRef   string
}

// GroupInfo holds parsed information about an API group.
type GroupInfo struct {
	GroupName    string
	GroupVersion string
	ModuleName   string
	Types        []TypeInfo
	Endpoints    []EndpointInfo // standalone endpoints from +openapi:endpoint
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

					// Skip unexported types (storage types handled separately)
					if !typeSpec.Name.IsExported() {
						continue
					}

					annotations := ParseAnnotations(genDecl.Doc)
					if len(annotations) == 0 {
						// Skip types without +openapi: annotations
						continue
					}

					// Extract description, paths, operation summaries, action summaries, and schema flag
					var description string
					var paths []string
					var schemaOnly bool
					opSummary := make(map[string]string)
					actionSummary := make(map[string]string)
					for _, ann := range annotations {
						switch {
						case ann.Key == "schema":
							schemaOnly = true
						case ann.Key == "description":
							if description == "" {
								description = ann.Value
							}
						case ann.Key == "path":
							if ann.Value != "" {
								paths = append(paths, ann.Value)
							}
						case strings.HasPrefix(ann.Key, "summary."):
							method := strings.TrimPrefix(ann.Key, "summary.")
							opSummary[method] = ann.Value
						case strings.HasPrefix(ann.Key, "action.") && strings.HasSuffix(ann.Key, ".summary"):
							actionName := strings.TrimPrefix(ann.Key, "action.")
							actionName = strings.TrimSuffix(actionName, ".summary")
							actionSummary[actionName] = ann.Value
						}
					}

					typeInfo := TypeInfo{
						Name:               typeSpec.Name.Name,
						Package:            pkg.Name,
						Annotations:        annotations,
						Description:        description,
						IsListType:         strings.HasSuffix(typeSpec.Name.Name, "List"),
						SchemaOnly:         schemaOnly,
						Paths:              paths,
						OperationSummary:   opSummary,
						ActionSummary:      actionSummary,
						CustomVerbSummary:  make(map[string]string),
						CustomVerbResponse: make(map[string]string),
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

	// Phase 2: Scan storage types, methods, and standalone functions
	// to collect operation-level annotations and merge into TypeInfo.
	mergeStorageAnnotations(group, pkgs)

	// Phase 3: Scan for standalone endpoint annotations (+openapi:endpoint)
	group.Endpoints = parseEndpointAnnotations(pkgs)

	if len(group.Types) == 0 {
		return nil, nil
	}

	return group, nil
}

// mergeStorageAnnotations scans for storage types (*Storage structs),
// their methods, and standalone functions with +openapi: annotations,
// then merges the derived paths and summaries into the corresponding TypeInfo.
func mergeStorageAnnotations(group *GroupInfo, pkgs map[string]*ast.Package) {
	// Build index: resource name → TypeInfo position
	typeIndex := make(map[string]int)
	for i, t := range group.Types {
		typeIndex[t.Name] = i
	}

	type storageInfo struct {
		resourceName    string
		derivedPath     string
		qualifiedPrefix string
		extraPaths      []string
	}
	storageTypes := make(map[string]*storageInfo)

	for _, pkg := range pkgs {
		// Scan all struct types ending with "Storage"
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
					name := typeSpec.Name.Name
					if !strings.HasSuffix(name, "Storage") || typeSpec.Name.IsExported() {
						continue
					}
					baseName := strings.TrimSuffix(name, "Storage")
					segments := splitCamelCase(baseName)
					resource, path := deriveResourceAndPath(segments)

					var extraPaths []string
					var overrideResource string
					for _, ann := range ParseAnnotations(genDecl.Doc) {
						switch ann.Key {
						case "path":
							if ann.Value != "" {
								extraPaths = append(extraPaths, ann.Value)
							}
						case "resource":
							if ann.Value != "" {
								overrideResource = ann.Value
							}
						}
					}

					// If +openapi:resource is set, override the auto-derived resource name
					if overrideResource != "" {
						resource = overrideResource
					}

					// If extra paths are specified, use the first one as derived path
					// (override the auto-derived path which may be wrong)
					if len(extraPaths) > 0 {
						path = extraPaths[0]
					}
					prefix := pathToQualifiedPrefix(path)

					storageTypes[name] = &storageInfo{
						resourceName:    resource,
						derivedPath:     path,
						qualifiedPrefix: prefix,
						extraPaths:      extraPaths,
					}
				}
			}
		}

		// Merge storage paths into TypeInfo
		for _, st := range storageTypes {
			idx, ok := typeIndex[st.resourceName]
			if !ok {
				continue
			}
			group.Types[idx].Paths = appendUnique(group.Types[idx].Paths, st.derivedPath)
			for _, ep := range st.extraPaths {
				group.Types[idx].Paths = appendUnique(group.Types[idx].Paths, ep)
			}
		}

		// Scan methods for +openapi:summary= annotations
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				if funcDecl.Recv != nil {
					// Method: check if receiver is a storage type
					recvType := receiverTypeName(funcDecl.Recv)
					st, ok := storageTypes[recvType]
					if !ok {
						continue
					}
					op, ok := methodToOperation(funcDecl.Name.Name)
					if !ok {
						continue
					}
					annotations := ParseAnnotations(funcDecl.Doc)
					if len(annotations) == 0 {
						continue
					}
					idx, ok := typeIndex[st.resourceName]
					if !ok {
						continue
					}
					for _, ann := range annotations {
						switch {
						case ann.Key == "summary":
							key := operationKey(st.qualifiedPrefix, op)
							group.Types[idx].OperationSummary[key] = ann.Value
						case strings.HasPrefix(ann.Key, "summary."):
							qualifier := strings.TrimPrefix(ann.Key, "summary.")
							key := qualifier + "." + op
							group.Types[idx].OperationSummary[key] = ann.Value
						}
					}
				} else {
					// Standalone function: check for +openapi:action= or +openapi:customverb=
					annotations := ParseAnnotations(funcDecl.Doc)
					if len(annotations) == 0 {
						continue
					}
					var actionName, customVerb, summary, resource, response string
					for _, ann := range annotations {
						switch ann.Key {
						case "action":
							actionName = ann.Value
						case "customverb":
							customVerb = ann.Value
						case "summary":
							summary = ann.Value
						case "resource":
							resource = ann.Value
						case "response":
							response = ann.Value
						}
					}
					if actionName != "" && resource != "" {
						if idx, ok := typeIndex[resource]; ok && summary != "" {
							group.Types[idx].ActionSummary[actionName] = summary
						}
					}
					if customVerb != "" && resource != "" {
						if idx, ok := typeIndex[resource]; ok {
							if summary != "" {
								group.Types[idx].CustomVerbSummary[customVerb] = summary
							}
							if response != "" {
								group.Types[idx].CustomVerbResponse[customVerb] = response
							}
						}
					}
				}
			}
		}
	}
}

// splitCamelCase splits a camelCase name into lowercase segments.
// e.g., "workspaceUser" → ["workspace", "user"]
func splitCamelCase(s string) []string {
	var segments []string
	start := 0
	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			segments = append(segments, strings.ToLower(s[start:i]))
			start = i
		}
	}
	segments = append(segments, strings.ToLower(s[start:]))
	return segments
}

// deriveResourceAndPath derives the resource type name and API path from
// camelCase-split storage name segments.
// e.g., ["workspace", "user"] → ("User", "/workspaces/{workspaceId}/users")
func deriveResourceAndPath(segments []string) (resource, path string) {
	if len(segments) == 0 {
		return "", ""
	}
	last := segments[len(segments)-1]
	resource = strings.ToUpper(last[:1]) + last[1:]

	var parts []string
	for _, seg := range segments[:len(segments)-1] {
		parts = append(parts, seg+"s", "{"+seg+"Id}")
	}
	parts = append(parts, last+"s")
	path = "/" + strings.Join(parts, "/")
	return
}

// pathToQualifiedPrefix converts a path to a dotted qualifier prefix.
// e.g., "/workspaces/{workspaceId}/users" → "workspaces.users"
// Single-segment paths like "/users" return "".
func pathToQualifiedPrefix(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var segments []string
	for _, p := range parts {
		if p != "" && !strings.HasPrefix(p, "{") {
			segments = append(segments, p)
		}
	}
	if len(segments) <= 1 {
		return ""
	}
	return strings.Join(segments, ".")
}

// operationKey builds the OperationSummary map key from a qualified prefix and operation.
func operationKey(qualifiedPrefix, op string) string {
	if qualifiedPrefix == "" {
		return op
	}
	return qualifiedPrefix + "." + op
}

// receiverTypeName extracts the type name from a method receiver.
func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// methodToOperation maps Go method names to OpenAPI operation keys.
func methodToOperation(name string) (string, bool) {
	ops := map[string]string{
		"List":             "list",
		"Create":           "create",
		"Get":              "get",
		"Update":           "update",
		"Patch":            "patch",
		"Delete":           "delete",
		"DeleteCollection": "deleteCollection",
	}
	op, ok := ops[name]
	return op, ok
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

// parseEndpointAnnotations scans all functions for +openapi:endpoint annotations
// and builds EndpointInfo entries from them.
func parseEndpointAnnotations(pkgs map[string]*ast.Package) []EndpointInfo {
	var endpoints []EndpointInfo

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv != nil {
					continue
				}

				annotations := ParseAnnotations(funcDecl.Doc)
				if len(annotations) == 0 {
					continue
				}

				// Check if this function has +openapi:endpoint
				isEndpoint := false
				for _, ann := range annotations {
					if ann.Key == "endpoint" {
						isEndpoint = true
						break
					}
				}
				if !isEndpoint {
					continue
				}

				ep := EndpointInfo{}
				var responses []EndpointResponse
				for _, ann := range annotations {
					switch {
					case ann.Key == "path":
						ep.Path = ann.Value
					case ann.Key == "method":
						ep.Method = strings.ToUpper(ann.Value)
					case ann.Key == "summary":
						ep.Summary = ann.Value
					case ann.Key == "description":
						ep.Description = ann.Value
					case ann.Key == "tag":
						ep.Tag = ann.Value
					case ann.Key == "operationId":
						ep.OperationID = ann.Value
					case ann.Key == "requestBody.contentType":
						if ep.RequestBody == nil {
							ep.RequestBody = &EndpointBody{}
						}
						ep.RequestBody.ContentType = ann.Value
					case ann.Key == "requestBody.schema":
						if ep.RequestBody == nil {
							ep.RequestBody = &EndpointBody{}
						}
						ep.RequestBody.SchemaRef = ann.Value
					case strings.HasPrefix(ann.Key, "response."):
						// +openapi:response.200.description=OK
						// +openapi:response.200.contentType=application/json
						// +openapi:response.200.schema=TokenResponse
						rest := strings.TrimPrefix(ann.Key, "response.")
						code, field, ok := strings.Cut(rest, ".")
						if !ok {
							continue
						}
						// Find or create response entry
						var resp *EndpointResponse
						for i := range responses {
							if responses[i].StatusCode == code {
								resp = &responses[i]
								break
							}
						}
						if resp == nil {
							responses = append(responses, EndpointResponse{StatusCode: code})
							resp = &responses[len(responses)-1]
						}
						switch field {
						case "description":
							resp.Description = ann.Value
						case "contentType":
							resp.ContentType = ann.Value
						case "schema":
							resp.SchemaRef = ann.Value
						}
					}
				}

				ep.Responses = responses
				if ep.Path != "" && ep.Method != "" {
					endpoints = append(endpoints, ep)
				}
			}
		}
	}

	return endpoints
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
