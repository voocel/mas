package typed

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/voocel/mas/llm"
)

// CleanTypeName returns a clean type name without package prefix.
func CleanTypeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return "Response"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()
	if name == "" {
		return "Response"
	}
	return name
}

// SchemaFromType generates a JSON Schema from a Go type.
func SchemaFromType[T any]() map[string]interface{} {
	var zero T
	return generateSchema(reflect.TypeOf(zero))
}

// ResponseFormatFromType creates a ResponseFormat for structured output.
func ResponseFormatFromType[T any](name string) *llm.ResponseFormat {
	schema := SchemaFromType[T]()
	strict := true
	return &llm.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]interface{}{
			"name":   name,
			"schema": schema,
			"strict": true,
		},
		Strict: &strict,
	}
}

// ParseResponse parses JSON response into typed struct.
func ParseResponse[T any](content string) (T, error) {
	var result T
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return result, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

var timeType = reflect.TypeOf(time.Time{})

func generateSchema(t reflect.Type) map[string]interface{} {
	if t == nil {
		return map[string]interface{}{"type": "object"}
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle time.Time specially
	if t == timeType {
		return map[string]interface{}{
			"type":   "string",
			"format": "date-time",
		}
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": generateSchema(t.Elem()),
		}
	case reflect.Map:
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": generateSchema(t.Elem()),
		}
	case reflect.Struct:
		return generateStructSchema(t)
	default:
		return map[string]interface{}{"type": "string"}
	}
}

func generateStructSchema(t reflect.Type) map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		isOptional := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					isOptional = true
				}
			}
		}

		fieldSchema := generateSchema(field.Type)
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		}

		if enum := field.Tag.Get("enum"); enum != "" {
			fieldSchema["enum"] = strings.Split(enum, ",")
		}

		properties[fieldName] = fieldSchema

		if !isOptional {
			required = append(required, fieldName)
		}
	}

	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}
