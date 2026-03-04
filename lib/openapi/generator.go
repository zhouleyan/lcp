package openapi

import (
	"fmt"
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

	for _, group := range groups {
		g.processGroup(doc, group)
	}

	return doc
}

func (g *Generator) processGroup(doc *Document, group GroupInfo) {
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

	// Detect sub-resource types: types whose Spec counterpart exists (e.g., NamespaceMemberSpec)
	subResourceTypes := map[string]bool{}
	for _, t := range group.Types {
		if t.IsListType {
			continue
		}
		if specTypes[t.Name] {
			continue
		}
		// A type is a sub-resource if it contains a parent resource name as prefix
		// e.g., "NamespaceMember" contains "Namespace" => sub-resource
		for _, other := range group.Types {
			if other.Name != t.Name && !other.IsListType && !specTypes[other.Name] &&
				strings.HasPrefix(t.Name, other.Name) && t.Name != other.Name {
				subResourceTypes[t.Name] = true
			}
		}
	}

	for _, t := range group.Types {
		// Add schema to components
		schema := g.typeToSchema(t)
		doc.Components.Schemas[t.Name] = schema

		// Generate paths only for main resource types (not List, Spec, Meta, or sub-resources)
		if !t.IsListType && !specTypes[t.Name] && !subResourceTypes[t.Name] {
			resourceName := strings.ToLower(t.Name) + "s"
			g.generatePaths(doc, basePath, resourceName, t.Name, group.GroupVersion)
		}
	}
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

func (g *Generator) generatePaths(doc *Document, basePath, resourceName, typeName, version string) {
	ref := &Schema{Ref: fmt.Sprintf("#/components/schemas/%s", typeName)}
	listRef := &Schema{Ref: fmt.Sprintf("#/components/schemas/%sList", typeName)}

	collectionPath := basePath + "/" + resourceName
	itemPath := collectionPath + "/{id}"

	tag := typeName

	// Collection operations
	pathItem := getOrCreatePathItem(doc, collectionPath)
	pathItem.Get = &Operation{
		Summary:     fmt.Sprintf("List %s", resourceName),
		OperationID: fmt.Sprintf("list%s", typeName),
		Tags:        []string{tag},
		Parameters: []Parameter{
			{Name: "page", In: "query", Schema: &Schema{Type: "integer"}},
			{Name: "pageSize", In: "query", Schema: &Schema{Type: "integer"}},
			{Name: "sortBy", In: "query", Schema: &Schema{Type: "string"}},
			{Name: "sortOrder", In: "query", Schema: &Schema{Type: "string", Enum: []string{"asc", "desc"}}},
		},
		Responses: map[string]*Response{
			"200": {
				Description: "OK",
				Content: map[string]MediaType{
					"application/json": {Schema: listRef},
				},
			},
		},
	}
	pathItem.Post = &Operation{
		Summary:     fmt.Sprintf("Create a %s", typeName),
		OperationID: fmt.Sprintf("create%s", typeName),
		Tags:        []string{tag},
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

	// Item operations
	itemPathItem := getOrCreatePathItem(doc, itemPath)
	idParam := Parameter{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"}}

	itemPathItem.Get = &Operation{
		Summary:     fmt.Sprintf("Get a %s", typeName),
		OperationID: fmt.Sprintf("get%s", typeName),
		Tags:        []string{tag},
		Parameters:  []Parameter{idParam},
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
		Summary:     fmt.Sprintf("Update a %s", typeName),
		OperationID: fmt.Sprintf("update%s", typeName),
		Tags:        []string{tag},
		Parameters:  []Parameter{idParam},
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
		Summary:     fmt.Sprintf("Patch a %s", typeName),
		OperationID: fmt.Sprintf("patch%s", typeName),
		Tags:        []string{tag},
		Parameters:  []Parameter{idParam},
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
		Summary:     fmt.Sprintf("Delete a %s", typeName),
		OperationID: fmt.Sprintf("delete%s", typeName),
		Tags:        []string{tag},
		Parameters:  []Parameter{idParam},
		Responses: map[string]*Response{
			"204": {Description: "No Content"},
		},
	}
}

func getOrCreatePathItem(doc *Document, path string) *PathItem {
	if item, ok := doc.Paths[path]; ok {
		return item
	}
	item := &PathItem{}
	doc.Paths[path] = item
	return item
}
