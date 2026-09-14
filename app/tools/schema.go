//go:build windows || darwin

package tools

import (
	"encoding/json"
	"fmt"

	"github.com/ollama/ollama/api"
)

// namedProperty pairs a parameter's name with its definition.
//
// Declaration order is kept: api.ToolPropertiesMap preserves insertion order,
// and that order is what the model reads.
type namedProperty struct {
	Name     string
	Property api.ToolProperty
}

// toolProperties builds an ordered property map from the declarations given.
func toolProperties(properties []namedProperty) *api.ToolPropertiesMap {
	ordered := api.NewToolPropertiesMap()
	for _, entry := range properties {
		ordered.Set(entry.Name, entry.Property)
	}
	return ordered
}

// toolSchemaMap renders a set of parameters as the plain map that the Tool
// interface's Schema method returns.
//
// Tools that hold a faithful api.ToolFunction derive the map from it, never the
// other way round. Deriving in this direction cannot lose anything, because the
// map is built from the definition the model actually receives and the two
// therefore cannot disagree. The arrangement it replaces ran uphill: a
// hand-written map was rebuilt into a definition by toolFunctionFromSchema,
// which carries only a property's type and description, so anything the map
// said beyond that was written down and then thrown away.
func toolSchemaMap(parameters api.ToolFunctionParameters) (map[string]any, error) {
	encoded, err := json.Marshal(parameters)
	if err != nil {
		return nil, fmt.Errorf("encode parameters: %w", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(encoded, &schema); err != nil {
		return nil, fmt.Errorf("decode parameters: %w", err)
	}
	return schema, nil
}

// schemaOf is the Schema implementation shared by the first-party tools, whose
// parameters are declared as an api.ToolFunction in this package.
//
// The error is dropped because Tool.Schema has nowhere to report one; a nil map
// is returned instead, which is what these tools already did when their
// hand-written JSON failed to parse. The input is a static literal here, so the
// only way it can fail is a programming error, and the definitions are covered
// by TestFirstPartyToolDefinitionsAndSchemaMapsAgree.
func schemaOf(fn api.ToolFunction) map[string]any {
	schema, err := toolSchemaMap(fn.Parameters)
	if err != nil {
		return nil
	}
	return schema
}
