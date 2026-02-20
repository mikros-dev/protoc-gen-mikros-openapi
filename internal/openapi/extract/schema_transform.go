package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
)

// transformRules defines schema transformation callbacks.
type transformRules struct {
	// TransformRef is a function called for every schema node that has a
	// non-empty Ref field. It should return the transformed reference.
	TransformRef func(string) string

	// TransformPropertyName is a function called for every property in a schema
	// when the parent is an object schema. It should return the transformed property
	// name.
	TransformPropertyName func(parent *spec.Schema, name string, property *spec.Schema) (string, error)
}

// transformSchema recursively transforms schema nodes in place according to
// the given rules.
func transformSchema(schema *spec.Schema, rules transformRules) error {
	if schema == nil {
		return nil
	}

	transformRef(schema, rules)

	if err := transformChildren(schema, rules); err != nil {
		return err
	}

	return transformProperties(schema, rules)
}

func transformRef(schema *spec.Schema, rules transformRules) {
	if schema.Ref == "" || rules.TransformRef == nil {
		return
	}

	schema.Ref = rules.TransformRef(schema.Ref)
}

func transformChildren(schema *spec.Schema, rules transformRules) error {
	// Keep the traversal logic centralized so we can let transformSchema simple.
	children := []*spec.Schema{
		schema.Items,
		schema.AdditionalProperties,
	}

	for _, child := range children {
		if child == nil {
			continue
		}

		if err := transformSchema(child, rules); err != nil {
			return err
		}
	}

	for _, node := range schema.AnyOf {
		if err := transformSchema(node, rules); err != nil {
			return err
		}
	}

	return nil
}

func transformProperties(schema *spec.Schema, rules transformRules) error {
	if len(schema.Properties) == 0 {
		return nil
	}

	// The fast path. No renaming, just recurse into properties.
	if rules.TransformPropertyName == nil {
		for _, property := range schema.Properties {
			if err := transformSchema(property, rules); err != nil {
				return err
			}
		}

		return nil
	}

	// The renaming path. Rebuild the map.
	var (
		properties = make(map[string]*spec.Schema, len(schema.Properties))
		renamed    = make(map[string]string, len(schema.Properties))
	)

	for name, property := range schema.Properties {
		if err := transformSchema(property, rules); err != nil {
			return err
		}

		transformedName, err := rules.TransformPropertyName(schema, name, property)
		if err != nil {
			return err
		}

		renamed[name] = transformedName
		properties[transformedName] = property
	}

	schema.Properties = properties
	schema.RequiredProperties = transformRequiredProperties(schema.RequiredProperties, renamed)

	return nil
}

func transformRequiredProperties(required []string, renamed map[string]string) []string {
	if len(required) == 0 {
		return required
	}

	out := make([]string, len(required))
	for i, name := range required {
		if n, ok := renamed[name]; ok {
			out[i] = n
			continue
		}

		out[i] = name
	}

	return out
}
