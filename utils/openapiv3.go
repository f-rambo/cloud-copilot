package utils

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Info represents the metadata of the API.
type Info struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

// Parameter represents a query parameter in an operation.
type Parameter struct {
	Name   string `yaml:"name"`
	In     string `yaml:"in"`
	Schema struct {
		Type   string `yaml:"type"`
		Format string `yaml:"format,omitempty"`
	} `yaml:"schema"`
}

type SchemaRef struct {
	Ref string `yaml:"$ref,omitempty"`
}

type MediaType struct {
	Schema SchemaRef `yaml:"schema,omitempty"`
}

type Content struct {
	JSON MediaType `yaml:"application/json,omitempty"`
}

type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty"`
	Ref         string               `yaml:"$ref,omitempty"`
}

type Responses map[string]Response

type RequestBody struct {
	Content  map[string]MediaType `yaml:"content"`
	Required bool                 `yaml:"required"`
}

// Operation represents an API operation (e.g., GET, POST).
type Operation struct {
	Tags        []string     `yaml:"tags"`
	OperationID string       `yaml:"operationId"`
	Parameters  []Parameter  `yaml:"parameters,omitempty"`
	Responses   Responses    `yaml:"responses,omitempty"`
	RequestBody *RequestBody `yaml:"requestBody,omitempty"`
	Description string       `yaml:"description,omitempty"`
}

// PathItem represents the operations available on a single path.
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// Paths represents all paths in the API.
type Paths map[string]PathItem

// Schema represents a schema definition in components.
type Schema struct {
	Type                 string            `yaml:"type"`
	Properties           map[string]Schema `yaml:"properties,omitempty"`
	Items                *Schema           `yaml:"items,omitempty"`
	Ref                  string            `yaml:"$ref,omitempty"`
	Format               string            `yaml:"format,omitempty"`
	AdditionalProperties *Schema           `yaml:"additionalProperties,omitempty"`
	Description          string            `yaml:"description,omitempty"`
}

// Components holds reusable schemas.
type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

// Tag represents a tag with optional description.
type Tag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// OpenAPISpec represents the full OpenAPI specification.
type OpenAPISpec struct {
	OpenAPI    string     `yaml:"openapi"`
	Info       Info       `yaml:"info"`
	Paths      Paths      `yaml:"paths"`
	Components Components `yaml:"components,omitempty"`
	Tags       []Tag      `yaml:"tags,omitempty"`
}

func (s *Schema) IsArray() bool {
	return s.Type == "array" || (s.Items != nil && s.Items.Type == "array")
}

func (s *Schema) IsObject() bool {
	return s.Type == "object" || (s.Properties != nil && len(s.Properties) > 0)
}

func (s *Schema) IsString() bool {
	return s.Type == "string" || (s.Format != "" && s.Format != "object" && s.Format != "array")
}

func (s *Schema) IsNumber() bool {
	return s.Type == "number" || s.Type == "integer" || (s.Format != "" && (s.Format == "float" || s.Format == "double"))
}

// GetSchemaParameters returns all properties of a schema referenced by $ref.
func GetSchemaParameters(ref string, components Components) (map[string]Schema, error) {
	if !strings.HasPrefix(ref, "#/components/schemas/") {
		return nil, fmt.Errorf("invalid ref: %s", ref)
	}

	schemaName := strings.TrimPrefix(ref, "#/components/schemas/")

	schema, exists := components.Schemas[schemaName]
	if !exists {
		return nil, errors.New("schema not found in components")
	}

	return getPropertiesRecursive(schema, components)
}

// getPropertiesRecursive handles nested schemas and resolves $ref recursively.
func getPropertiesRecursive(schema Schema, components Components) (map[string]Schema, error) {
	if schema.Properties != nil {
		return schema.Properties, nil
	}

	if schema.Ref != "" {
		return GetSchemaParameters(schema.Ref, components)
	}

	return nil, errors.New("schema does not contain properties or valid $ref")
}
