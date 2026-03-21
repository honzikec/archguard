package brief

import (
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	schemaOnce     sync.Once
	compiledSchema *jsonschema.Schema
	schemaInitErr  error
)

func validateSchemaDocument(raw any) error {
	schema, err := loadCompiledSchema()
	if err != nil {
		return err
	}
	if err := schema.Validate(normalizeYAMLDocument(raw)); err != nil {
		return fmt.Errorf("brief schema validation failed: %w", err)
	}
	return nil
}

func loadCompiledSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		resource, err := jsonschema.UnmarshalJSON(strings.NewReader(briefSchemaV1))
		if err != nil {
			schemaInitErr = fmt.Errorf("invalid embedded brief schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("archguard.brief.schema.json", resource); err != nil {
			schemaInitErr = fmt.Errorf("failed to add embedded brief schema: %w", err)
			return
		}
		compiledSchema, schemaInitErr = compiler.Compile("archguard.brief.schema.json")
		if schemaInitErr != nil {
			schemaInitErr = fmt.Errorf("failed to compile brief schema: %w", schemaInitErr)
		}
	})
	return compiledSchema, schemaInitErr
}

func normalizeYAMLDocument(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalizeYAMLDocument(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = normalizeYAMLDocument(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalizeYAMLDocument(x[i])
		}
		return out
	default:
		return v
	}
}
