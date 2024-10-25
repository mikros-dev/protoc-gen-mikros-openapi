package openapi

import (
	"slices"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/protobuf/reflect/protoreflect"

	openapipb "github.com/mikros-dev/protoc-gen-openapi/openapi"
)

type Schema struct {
	Minimum              int                `yaml:"minimum,omitempty"`
	Maximum              int                `yaml:"maximum,omitempty"`
	Type                 string             `yaml:"type,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	Ref                  string             `yaml:"$ref,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Example              string             `yaml:"example,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`
	Enum                 []string           `yaml:"enum,omitempty"`
	Required             []string           `yaml:"required,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty"`
	AnyOf                []*Schema          `yaml:"anyOf,omitempty"`

	schemaType SchemaType
	required   bool
}

func newRefSchema(field *protobuf.Field, refDestination string, pkg *protobuf.Protobuf) *Schema {
	schema := newSchemaFromProtobufField(field, pkg)

	if schema.Type == SchemaType_Array.String() {
		schema.Items = &Schema{
			Ref: refComponentsSchemas + refDestination,
		}
	}

	if schema.Type != SchemaType_Array.String() {
		schema.Type = "" // Clears the type
		schema.Ref = refComponentsSchemas + refDestination
	}

	return schema
}

func newSchemaFromProtobufField(field *protobuf.Field, pkg *protobuf.Protobuf) *Schema {
	var (
		properties = openapipb.LoadFieldExtensions(field.Proto)
		schema     = &Schema{
			Type: schemaTypeFromProtobufField(field).String(),
		}
	)

	if properties != nil {
		schema.required = properties.GetRequired()
		schema.Example = properties.GetExample()
		schema.Description = properties.GetDescription()
	}

	if field.IsTimestamp() {
		schema.Format = "date-time"
	}

	if field.IsEnum() {
		schema.Enum = getEnumValues(field, pkg)
	}

	if field.IsProtoStruct() {
		// metadata
		schema.Type = SchemaType_Object.String()
		schema.AdditionalProperties = &Schema{}
	}

	if field.IsProtoValue() {
		// interface
		schema.Type = SchemaType_Object.String()
		for _, t := range []SchemaType{SchemaType_String, SchemaType_Integer, SchemaType_Number, SchemaType_Bool, SchemaType_Object, SchemaType_Array} {
			schema.AnyOf = append(schema.AnyOf, &Schema{
				Type: t.String(),
			})
		}
	}

	if field.IsMap() {
		// Map should always have keys as string, because JSON does not support
		// other types as keys.
		schema.Type = SchemaType_Object.String()
		schema.AdditionalProperties = getMapSchema(field)
	}

	if field.IsArray() {
		schema.Type = SchemaType_Array.String()
		if schema.Items == nil {
			schema.Items = &Schema{
				Type: schemaTypeFromProtobufField(field).String(),
			}
		}
	}

	return schema
}

func getMapSchema(field *protobuf.Field) *Schema {
	schema := &Schema{
		Type: schemaTypeFromMapType(field.MapValueTypeKind()).String(),
	}

	if field.MapValueTypeKind() == protoreflect.MessageKind || field.MapValueTypeKind() == protoreflect.EnumKind {
		schema.Type = ""
		schema.Ref = refComponentsSchemas + trimPackageName(field.MapValueTypeName())
	}

	return schema
}

func schemaTypeFromMapType(mapType protoreflect.Kind) SchemaType {
	switch mapType {
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return SchemaType_Number
	case protoreflect.MessageKind:
		return SchemaType_Object
	case protoreflect.EnumKind, protoreflect.StringKind, protoreflect.BytesKind:
		return SchemaType_String
	case protoreflect.BoolKind:
		return SchemaType_Bool
	default:
	}

	return SchemaType_Integer
}

func getEnumValues(field *protobuf.Field, pkg *protobuf.Protobuf) []string {
	var (
		enums       []*protobuf.Enum
		packageName = getPackageName(field.TypeName)
		values      []string
	)

	if packageName == pkg.PackageName {
		// Get values from local module enum
		enums = pkg.Enums
	}
	if packageName != pkg.PackageName {
		// Or look for them in foreign packages.
		enums = loadForeignEnums(field.TypeName, pkg)
	}

	index := slices.IndexFunc(enums, func(enum *protobuf.Enum) bool {
		return enum.Name == trimPackageName(field.TypeName)
	})
	if index != -1 {
		for _, e := range enums[index].Values {
			values = append(values, e.ProtoName)
		}
	}

	return values
}

func loadForeignEnums(enumType string, pkg *protobuf.Protobuf) []*protobuf.Enum {
	var (
		foreignPackage = getPackageName(enumType)
		enums          []*protobuf.Enum
	)

	// Load foreign enums
	for _, f := range pkg.Files {
		if f.Proto.GetPackage() == foreignPackage {
			enums = protobuf.ParseEnumsFromFile(f)
			break
		}
	}

	return enums
}

func (s *Schema) IsRequired() bool {
	return s.required
}

func (s *Schema) HasAdditionalProperties() bool {
	return s.AdditionalProperties != nil && s.AdditionalProperties != &Schema{}
}

func (s *Schema) GetAdditionalPropertySchemas(field *protobuf.Field, pkg *protobuf.Protobuf) (map[string]*Schema, error) {
	if field.MapValueTypeKind() == protoreflect.MessageKind {
		return getMessageAdditionalSchema(field, pkg)
	}

	if field.MapValueTypeKind() == protoreflect.EnumKind {
		return map[string]*Schema{
			trimPackageName(field.MapValueTypeName()): getEnumAdditionalSchema(field, pkg),
		}, nil
	}

	return nil, nil
}

func getMessageAdditionalSchema(field *protobuf.Field, pkg *protobuf.Protobuf) (map[string]*Schema, error) {
	var (
		packageName = getPackageName(field.MapValueTypeName())
		messages    []*protobuf.Message
	)

	if packageName == pkg.PackageName {
		messages = pkg.Messages
	}
	if packageName != pkg.PackageName {
		// find foreign message
		m, err := loadForeignMessages(field.MapValueTypeName(), pkg)
		if err != nil {
			return nil, err
		}
		messages = m
	}

	// We expect this message to have no internal message fields, because
	// we won't dive into them.
	index := slices.IndexFunc(messages, func(msg *protobuf.Message) bool {
		return msg.Name == trimPackageName(field.MapValueTypeName())
	})
	if index != -1 {
		return getMessageSchemas(messages[index], pkg, nil, nil, nil)
	}

	return nil, nil
}

func getEnumAdditionalSchema(field *protobuf.Field, pkg *protobuf.Protobuf) *Schema {
	var (
		packageName = getPackageName(field.MapValueTypeName())
		enums       []*protobuf.Enum
		schema      = &Schema{
			Type: SchemaType_String.String(),
		}
	)

	if packageName == pkg.PackageName {
		// Get values from local module enum
		enums = pkg.Enums
	}
	if packageName != pkg.PackageName {
		// Or look for them in foreign packages.
		enums = loadForeignEnums(field.MapValueTypeName(), pkg)
	}

	index := slices.IndexFunc(enums, func(enum *protobuf.Enum) bool {
		return enum.Name == trimPackageName(field.MapValueTypeName())
	})
	if index != -1 {
		for _, e := range enums[index].Values {
			schema.Enum = append(schema.Enum, e.ProtoName)
		}
	}

	return schema
}
