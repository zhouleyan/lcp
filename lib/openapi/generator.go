package openapi

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Generator builds an OpenAPI 3.0 specification from parsed API groups.
type Generator struct {
	info Info
}

// NewGenerator creates a new OpenAPI generator with the given API info.
func NewGenerator(title, description, version string) *Generator {
	return &Generator{
		info: Info{
			Title:       title,
			Description: description,
			Version:     version,
		},
	}
}

// Generate builds a complete OpenAPI document from the parsed groups.
func (g *Generator) Generate(groups []GroupInfo) *Document {
	doc := &Document{
		OpenAPI: "3.0.3",
		Info:    g.info,
		Paths:   make(map[string]*PathItem),
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}

	// Collect tags per module for x-tagGroups
	moduleTags := make(map[string][]Tag)

	for _, group := range groups {
		tags := g.processGroup(doc, group)
		moduleTags[group.ModuleName] = append(moduleTags[group.ModuleName], tags...)
	}

	// Build document-level Tags and XTagGroups
	for _, moduleName := range sortedKeys(moduleTags) {
		tags := moduleTags[moduleName]
		var tagNames []string
		for _, t := range tags {
			doc.Tags = append(doc.Tags, t)
			tagNames = append(tagNames, t.Name)
		}
		doc.XTagGroups = append(doc.XTagGroups, TagGroup{
			Name: moduleName,
			Tags: tagNames,
		})
	}

	return doc
}

func (g *Generator) processGroup(doc *Document, group GroupInfo) []Tag {
	// Build base path
	var basePath string
	if group.GroupName == "" {
		basePath = "/api/" + group.GroupVersion
	} else {
		basePath = "/apis/" + group.GroupName + "/" + group.GroupVersion
	}

	// Collect all type names to identify which are Spec types (not standalone resources)
	specTypes := map[string]bool{}
	for _, t := range group.Types {
		if strings.HasSuffix(t.Name, "Spec") || strings.HasSuffix(t.Name, "Meta") {
			specTypes[t.Name] = true
		}
	}

	// Phase 1: Register all schemas in components
	for _, t := range group.Types {
		schema := g.typeToSchema(t)
		doc.Components.Schemas[t.Name] = schema
	}

	// Phase 2: Generate paths from +openapi:path annotations (or default)
	var tags []Tag
	for _, t := range group.Types {
		if t.IsListType || specTypes[t.Name] {
			continue
		}

		paths := t.Paths
		if len(paths) == 0 {
			// No annotation: derive default path from type name
			paths = []string{"/" + strings.ToLower(t.Name) + "s"}
		}

		tag := t.Name
		for _, p := range paths {
			g.generatePathsForResource(doc, basePath, p, t, group.GroupVersion, tag)
		}
		tags = append(tags, Tag{Name: t.Name, Description: t.Description})
	}

	return tags
}

func (g *Generator) typeToSchema(t TypeInfo) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	var required []string
	for _, field := range t.Fields {
		fieldSchema := goTypeToSchema(field.GoType)
		if field.Annotations.Description != "" {
			fieldSchema.Description = field.Annotations.Description
		}
		if field.Annotations.Format != "" {
			fieldSchema.Format = field.Annotations.Format
		}
		if len(field.Annotations.Enum) > 0 {
			fieldSchema.Enum = field.Annotations.Enum
		}
		if field.Annotations.Required {
			required = append(required, field.JSONName)
		}

		schema.Properties[field.JSONName] = fieldSchema
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}

func goTypeToSchema(goType string) *Schema {
	switch goType {
	case "string":
		return &Schema{Type: "string"}
	case "int", "int32", "int64":
		return &Schema{Type: "integer", Format: goType}
	case "float32":
		return &Schema{Type: "number", Format: "float"}
	case "float64":
		return &Schema{Type: "number", Format: "double"}
	case "bool":
		return &Schema{Type: "boolean"}
	default:
		if strings.HasPrefix(goType, "[]") {
			elemType := strings.TrimPrefix(goType, "[]")
			return &Schema{
				Type:  "array",
				Items: goTypeToSchema(elemType),
			}
		}
		if strings.HasPrefix(goType, "map[") {
			return &Schema{Type: "object"}
		}
		if strings.HasPrefix(goType, "*") {
			return goTypeToSchema(strings.TrimPrefix(goType, "*"))
		}
		// Assume it's a reference to another type
		cleanType := goType
		if idx := strings.LastIndex(goType, "."); idx >= 0 {
			cleanType = goType[idx+1:]
		}
		return &Schema{Ref: "#/components/schemas/" + cleanType}
	}
}

// generatePathsForResource generates collection and item paths for a resource
// at the given resourcePath. It extracts path parameters from the path template
// and uses a single tag for all operations.
func (g *Generator) generatePathsForResource(doc *Document, basePath, resourcePath string, typeInfo TypeInfo, version, tag string) {
	typeName := typeInfo.Name
	ref := &Schema{Ref: fmt.Sprintf("#/components/schemas/%s", typeName)}
	listRef := &Schema{Ref: fmt.Sprintf("#/components/schemas/%sList", typeName)}

	collectionPath := basePath + resourcePath
	idParam := deriveIDParam(resourcePath)
	itemPath := collectionPath + "/{" + idParam + "}"

	// Extract all {param} from the resource path as path parameters
	pathParams := extractPathParams(resourcePath)

	// Build a unique operation ID suffix from path segments to avoid collisions
	opSuffix := operationSuffix(resourcePath, typeName)

	// Helper to resolve operation summary: use annotation if present, otherwise default
	summary := func(op, defaultSummary string) string {
		if s, ok := typeInfo.OperationSummary[op]; ok {
			return s
		}
		return defaultSummary
	}

	// Qualified operation key for nested resources (e.g., "workspaces.namespaces.list")
	qualifiedSummary := func(op, defaultSummary string) string {
		// Try qualified key first (e.g., "workspaces.users.list"), then plain key
		qualifiedKey := resourcePath + "." + op
		// Normalize: strip leading /, replace {param}/ segments
		parts := strings.Split(strings.Trim(qualifiedKey, "/"), "/")
		var segments []string
		for _, p := range parts {
			if !strings.HasPrefix(p, "{") {
				segments = append(segments, p)
			}
		}
		qualified := strings.Join(segments, ".") // e.g. "workspaces.namespaces.users.list"
		if s, ok := typeInfo.OperationSummary[qualified]; ok {
			return s
		}
		return summary(op, defaultSummary)
	}

	// Collection operations
	pathItem := getOrCreatePathItem(doc, collectionPath)

	listParams := make([]Parameter, 0, len(pathParams)+4)
	for _, pp := range pathParams {
		listParams = append(listParams, Parameter{
			Name: pp, In: "path", Required: true, Schema: &Schema{Type: "string"},
		})
	}
	listParams = append(listParams,
		Parameter{Name: "page", In: "query", Schema: &Schema{Type: "integer"}},
		Parameter{Name: "pageSize", In: "query", Schema: &Schema{Type: "integer"}},
		Parameter{Name: "sortBy", In: "query", Schema: &Schema{Type: "string"}},
		Parameter{Name: "sortOrder", In: "query", Schema: &Schema{Type: "string", Enum: []string{"asc", "desc"}}},
	)

	pathItem.Get = &Operation{
		Summary:     qualifiedSummary("list", fmt.Sprintf("List %s", lastSegment(resourcePath))),
		OperationID: fmt.Sprintf("list%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  listParams,
		Responses: map[string]*Response{
			"200": {
				Description: "OK",
				Content: map[string]MediaType{
					"application/json": {Schema: listRef},
				},
			},
		},
	}

	createParams := make([]Parameter, 0, len(pathParams))
	for _, pp := range pathParams {
		createParams = append(createParams, Parameter{
			Name: pp, In: "path", Required: true, Schema: &Schema{Type: "string"},
		})
	}
	pathItem.Post = &Operation{
		Summary:     qualifiedSummary("create", fmt.Sprintf("Create a %s", typeName)),
		OperationID: fmt.Sprintf("create%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  createParams,
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: ref},
			},
		},
		Responses: map[string]*Response{
			"201": {
				Description: "Created",
				Content: map[string]MediaType{
					"application/json": {Schema: ref},
				},
			},
		},
	}

	// Delete collection
	pathItem.Delete = &Operation{
		Summary:     qualifiedSummary("deleteCollection", fmt.Sprintf("Batch delete %s", lastSegment(resourcePath))),
		OperationID: fmt.Sprintf("deleteCollection%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  createParams,
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"ids": {Type: "array", Items: &Schema{Type: "string"}},
					},
				}},
			},
		},
		Responses: map[string]*Response{
			"200": {Description: "OK"},
		},
	}

	// Item operations
	itemPathItem := getOrCreatePathItem(doc, itemPath)

	itemParams := make([]Parameter, 0, len(pathParams)+1)
	for _, pp := range pathParams {
		itemParams = append(itemParams, Parameter{
			Name: pp, In: "path", Required: true, Schema: &Schema{Type: "string"},
		})
	}
	itemParams = append(itemParams, Parameter{
		Name: idParam, In: "path", Required: true, Schema: &Schema{Type: "string"},
	})

	itemPathItem.Get = &Operation{
		Summary:     qualifiedSummary("get", fmt.Sprintf("Get a %s", typeName)),
		OperationID: fmt.Sprintf("get%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  itemParams,
		Responses: map[string]*Response{
			"200": {
				Description: "OK",
				Content: map[string]MediaType{
					"application/json": {Schema: ref},
				},
			},
		},
	}
	itemPathItem.Put = &Operation{
		Summary:     qualifiedSummary("update", fmt.Sprintf("Update a %s", typeName)),
		OperationID: fmt.Sprintf("update%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  itemParams,
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: ref},
			},
		},
		Responses: map[string]*Response{
			"200": {
				Description: "OK",
				Content: map[string]MediaType{
					"application/json": {Schema: ref},
				},
			},
		},
	}
	itemPathItem.Patch = &Operation{
		Summary:     qualifiedSummary("patch", fmt.Sprintf("Patch a %s", typeName)),
		OperationID: fmt.Sprintf("patch%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  itemParams,
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: ref},
			},
		},
		Responses: map[string]*Response{
			"200": {
				Description: "OK",
				Content: map[string]MediaType{
					"application/json": {Schema: ref},
				},
			},
		},
	}
	itemPathItem.Delete = &Operation{
		Summary:     qualifiedSummary("delete", fmt.Sprintf("Delete a %s", typeName)),
		OperationID: fmt.Sprintf("delete%s", opSuffix),
		Tags:        []string{tag},
		Parameters:  itemParams,
		Responses: map[string]*Response{
			"204": {Description: "No Content"},
		},
	}

	// Action operations
	for actionName, actionSummary := range typeInfo.ActionSummary {
		actionPath := itemPath + "/" + actionName
		actionPathItem := getOrCreatePathItem(doc, actionPath)
		actionPathItem.Post = &Operation{
			Summary:     actionSummary,
			OperationID: fmt.Sprintf("%s%s", toCamelCase(actionName), opSuffix),
			Tags:        []string{tag},
			Parameters:  itemParams,
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
			},
			Responses: map[string]*Response{
				"200": {Description: "OK"},
			},
		}
	}
}

var pathParamRegexp = regexp.MustCompile(`\{(\w+)\}`)

// extractPathParams returns all {param} names from a path template.
func extractPathParams(path string) []string {
	matches := pathParamRegexp.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, m := range matches {
		params = append(params, m[1])
	}
	return params
}

// deriveIDParam derives the item ID parameter name from the last segment
// of a resource path. e.g. "/users" -> "userId", "/namespaces/{namespaceId}/users" -> "userId"
func deriveIDParam(resourcePath string) string {
	seg := lastSegment(resourcePath)
	// Singularize: strip trailing "s"
	singular := strings.TrimSuffix(seg, "s")
	return singular + "Id"
}

// lastSegment returns the last path segment (without leading slash).
// e.g. "/namespaces/{namespaceId}/users" -> "users"
func lastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && !strings.HasPrefix(parts[i], "{") {
			return parts[i]
		}
	}
	return ""
}

// operationSuffix builds a unique operation ID suffix from the path.
// For top-level "/users" → "User", for nested "/namespaces/{namespaceId}/users" → "NamespaceUser".
func operationSuffix(resourcePath, typeName string) string {
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	// Collect non-param segments
	var segments []string
	for _, p := range parts {
		if p != "" && !strings.HasPrefix(p, "{") {
			segments = append(segments, p)
		}
	}
	if len(segments) <= 1 {
		return typeName
	}
	// Build prefix from parent segments: "namespaces" → "Namespace"
	var prefix string
	for _, seg := range segments[:len(segments)-1] {
		singular := strings.TrimSuffix(seg, "s")
		prefix += strings.ToUpper(singular[:1]) + singular[1:]
	}
	return prefix + typeName
}

func sortedKeys(m map[string][]Tag) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// toCamelCase converts "change-password" to "changePassword".
func toCamelCase(s string) string {
	parts := strings.Split(s, "-")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func getOrCreatePathItem(doc *Document, path string) *PathItem {
	if item, ok := doc.Paths[path]; ok {
		return item
	}
	item := &PathItem{}
	doc.Paths[path] = item
	return item
}
