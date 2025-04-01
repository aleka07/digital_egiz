package utils

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// JSONSchemaValidator handles validation against JSON schemas
type JSONSchemaValidator struct {
	schemas map[string]*gojsonschema.Schema
}

// NewJSONSchemaValidator creates a new JSONSchemaValidator
func NewJSONSchemaValidator() *JSONSchemaValidator {
	return &JSONSchemaValidator{
		schemas: make(map[string]*gojsonschema.Schema),
	}
}

// LoadSchema loads and compiles a JSON schema
func (v *JSONSchemaValidator) LoadSchema(name, schema string) error {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	compiledSchema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	v.schemas[name] = compiledSchema
	return nil
}

// ValidateAgainstSchema validates data against a named schema
func (v *JSONSchemaValidator) ValidateAgainstSchema(name string, data interface{}) error {
	schema, ok := v.schemas[name]
	if !ok {
		return fmt.Errorf("schema %s not found", name)
	}

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create document loader
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Check if valid
	if !result.Valid() {
		// Format validation errors
		var errorMessages string
		for i, err := range result.Errors() {
			if i > 0 {
				errorMessages += "; "
			}
			errorMessages += fmt.Sprintf("%s: %s", err.Field(), err.Description())
		}
		return fmt.Errorf("validation failed: %s", errorMessages)
	}

	return nil
}

// JSONSchemaBuilder helps build JSON schemas programmatically
type JSONSchemaBuilder struct {
	schema map[string]interface{}
}

// NewJSONSchemaBuilder creates a new JSONSchemaBuilder
func NewJSONSchemaBuilder() *JSONSchemaBuilder {
	return &JSONSchemaBuilder{
		schema: map[string]interface{}{
			"$schema":              "http://json-schema.org/draft-07/schema#",
			"type":                 "object",
			"additionalProperties": false,
			"properties":           map[string]interface{}{},
			"required":             []string{},
		},
	}
}

// SetTitle sets the schema title
func (b *JSONSchemaBuilder) SetTitle(title string) *JSONSchemaBuilder {
	b.schema["title"] = title
	return b
}

// SetDescription sets the schema description
func (b *JSONSchemaBuilder) SetDescription(description string) *JSONSchemaBuilder {
	b.schema["description"] = description
	return b
}

// AddProperty adds a property to the schema
func (b *JSONSchemaBuilder) AddProperty(name, propertyType string, required bool) *JSONSchemaBuilder {
	properties := b.schema["properties"].(map[string]interface{})
	properties[name] = map[string]interface{}{
		"type": propertyType,
	}

	if required {
		requiredProps := b.schema["required"].([]string)
		b.schema["required"] = append(requiredProps, name)
	}

	return b
}

// AddStringProperty adds a string property to the schema
func (b *JSONSchemaBuilder) AddStringProperty(name string, required bool) *JSONSchemaBuilder {
	return b.AddProperty(name, "string", required)
}

// AddNumberProperty adds a number property to the schema
func (b *JSONSchemaBuilder) AddNumberProperty(name string, required bool) *JSONSchemaBuilder {
	return b.AddProperty(name, "number", required)
}

// AddIntegerProperty adds an integer property to the schema
func (b *JSONSchemaBuilder) AddIntegerProperty(name string, required bool) *JSONSchemaBuilder {
	return b.AddProperty(name, "integer", required)
}

// AddBooleanProperty adds a boolean property to the schema
func (b *JSONSchemaBuilder) AddBooleanProperty(name string, required bool) *JSONSchemaBuilder {
	return b.AddProperty(name, "boolean", required)
}

// AddObjectProperty adds an object property to the schema
func (b *JSONSchemaBuilder) AddObjectProperty(name string, required bool) *JSONSchemaBuilder {
	properties := b.schema["properties"].(map[string]interface{})
	properties[name] = map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	if required {
		requiredProps := b.schema["required"].([]string)
		b.schema["required"] = append(requiredProps, name)
	}

	return b
}

// AddArrayProperty adds an array property to the schema
func (b *JSONSchemaBuilder) AddArrayProperty(name string, itemType string, required bool) *JSONSchemaBuilder {
	properties := b.schema["properties"].(map[string]interface{})
	properties[name] = map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": itemType,
		},
	}

	if required {
		requiredProps := b.schema["required"].([]string)
		b.schema["required"] = append(requiredProps, name)
	}

	return b
}

// Build returns the JSON schema as a string
func (b *JSONSchemaBuilder) Build() (string, error) {
	jsonBytes, err := json.MarshalIndent(b.schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	return string(jsonBytes), nil
}
