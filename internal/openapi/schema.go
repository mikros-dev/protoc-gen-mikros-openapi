package openapi

import (
	"errors"
	"slices"
	"strings"

	"github.com/juliangruber/go-intersect"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/settings"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

var (
	supportedSchemas = []SchemaType{
		SchemaTypeString,
		SchemaTypeInteger,
		SchemaTypeNumber,
		SchemaTypeBool,
		SchemaTypeObject,
		SchemaTypeArray,
	}
)

// Schema represents a swagger schema of a field/parameter/object.
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
	RequiredProperties   []string           `yaml:"required,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty"`
	AnyOf                []*Schema          `yaml:"anyOf,omitempty"`
	Message              *protobuf.Message  `yaml:"-"`

	schemaType SchemaType
	required   bool
	field      *protobuf.Field
}

func newRefSchema(
	field *protobuf.Field,
	refDestination string,
	pkg *protobuf.Protobuf,
	settings *settings.Settings,
) *Schema {
	schema := newSchemaFromProtobufField(field, pkg, settings)

	if schema.Type == SchemaTypeArray.String() {
		schema.Items = &Schema{
			Ref: refComponentsSchemas + refDestination,
		}
	}

	if schema.Type != SchemaTypeArray.String() {
		schema.Type = "" // Clears the type
		schema.Ref = refComponentsSchemas + refDestination
	}

	return schema
}

func newSchemaFromProtobufField(field *protobuf.Field, pkg *protobuf.Protobuf, settings *settings.Settings) *Schema {
	var (
		properties = mikros_openapi.LoadFieldExtensions(field.Proto)
		schema     = &Schema{
			Type:  schemaTypeFromProtobufField(field).String(),
			field: field, // Saves the field to be used later.
		}
	)

	if properties != nil {
		schema.required = properties.GetRequired()
		schema.Example = properties.GetExample()
		schema.Description = properties.GetDescription()
		schema.Format = protoFormatToSchemaFormat(properties.GetFormat())
	}

	if field.IsTimestamp() {
		// Timestamps are always formatted as date-time.
		schema.Format = "date-time"
	}

	if field.IsEnum() {
		schema.Enum = getEnumValues(field, pkg, settings)
	}

	// metadata
	if field.IsProtoStruct() {
		schema.Type = SchemaTypeObject.String()
		schema.AdditionalProperties = &Schema{}
	}

	// interface
	if field.IsProtoValue() {
		schema.Type = SchemaTypeObject.String()
		for _, t := range supportedSchemas {
			schema.AnyOf = append(schema.AnyOf, &Schema{
				Type: t.String(),
			})
		}
	}

	if field.IsMap() {
		// Map should always have keys as string, because JSON does not support
		// other types as keys.
		schema.Type = SchemaTypeObject.String()
		schema.AdditionalProperties = getMapSchema(field)
	}

	if field.IsArray() {
		schema.Type = SchemaTypeArray.String()
		if schema.Items == nil {
			schema.Items = &Schema{
				Type: schemaTypeFromProtobufField(field).String(),
			}
		}
	}

	return schema
}

func protoFormatToSchemaFormat(format mikros_openapi.PropertyFormat) string {
	switch format {
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_DATE_TIME:
		return "date-time"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_BINARY:
		return "binary"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_DOUBLE:
		return "double"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_FLOAT:
		return "float"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_INT32:
		return "int32"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_INT64:
		return "int64"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_BYTE:
		return "byte"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_DATE:
		return "date"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_PASSWORD:
		return "password"
	case mikros_openapi.PropertyFormat_PROPERTY_FORMAT_STRING:
		return "string"
	default:
		return ""
	}
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
		return SchemaTypeNumber
	case protoreflect.MessageKind:
		return SchemaTypeObject
	case protoreflect.EnumKind, protoreflect.StringKind, protoreflect.BytesKind:
		return SchemaTypeString
	case protoreflect.BoolKind:
		return SchemaTypeBool
	default:
	}

	return SchemaTypeInteger
}

func getEnumValues(field *protobuf.Field, pkg *protobuf.Protobuf, settings *settings.Settings) []string {
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
		var prefix string
		if settings.Enum.RemovePrefix {
			prefix = getEnumPrefix(enums[index])
		}

		for _, e := range enums[index].Values {
			if settings.Enum.RemoveUnspecifiedEntry {
				if strings.HasSuffix(e.ProtoName, "_UNSPECIFIED") {
					continue
				}
			}

			values = append(values, strings.TrimPrefix(e.ProtoName, prefix))
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

func getEnumPrefix(enum *protobuf.Enum) string {
	if len(enum.Values) <= 1 {
		return ""
	}

	return enumStringsIntersection(enum.Values[0].ProtoName, enum.Values[1].ProtoName)
}

func enumStringsIntersection(s1, s2 string) string {
	p1 := strings.Split(s1, "_")
	p2 := strings.Split(s2, "_")

	i := intersect.Simple(p1, p2)
	if len(i) == 0 {
		return ""
	}

	var parts []string
	for _, s := range i {
		parts = append(parts, s.(string))
	}

	return strings.Join(parts, "_") + "_"
}

// IsRequired returns true if the field is required.
func (s *Schema) IsRequired() bool {
	return s.required
}

// HasAdditionalProperties returns true if the field has additional properties.
func (s *Schema) HasAdditionalProperties() bool {
	return s.AdditionalProperties != nil && s.AdditionalProperties != &Schema{}
}

// GetAdditionalPropertySchemas returns additional properties schemas for the
// field.
func (s *Schema) GetAdditionalPropertySchemas(
	field *protobuf.Field,
	parser *MessageParser,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
) (map[string]*Schema, error) {
	if field.MapValueTypeKind() == protoreflect.MessageKind {
		return getMessageAdditionalSchema(field, parser, methodExtensions, httpCtx)
	}

	if field.MapValueTypeKind() == protoreflect.EnumKind {
		return map[string]*Schema{
			trimPackageName(field.MapValueTypeName()): getEnumAdditionalSchema(field, parser.Package),
		}, nil
	}

	return nil, nil
}

func getMessageAdditionalSchema(
	field *protobuf.Field,
	parser *MessageParser,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
) (map[string]*Schema, error) {
	var (
		packageName = getPackageName(field.MapValueTypeName())
		messages    []*protobuf.Message
	)

	if packageName == parser.Package.PackageName {
		messages = parser.Package.Messages
	}
	if packageName != parser.Package.PackageName {
		// find foreign message
		m, err := loadForeignMessages(field.MapValueTypeName(), parser.Package)
		if err != nil {
			return nil, err
		}
		messages = m
	}

	// We expect this message to have no internal message fields because
	// we won't dive into them.
	index := slices.IndexFunc(messages, func(msg *protobuf.Message) bool {
		return msg.Name == trimPackageName(field.MapValueTypeName())
	})
	if index != -1 {
		return parser.GetMessageSchemas(messages[index], methodExtensions, httpCtx)
	}

	return nil, nil
}

func loadForeignMessages(msgType string, pkg *protobuf.Protobuf) ([]*protobuf.Message, error) {
	var (
		foreignPackage = getPackageName(msgType)
		messages       []*protobuf.Message
	)

	// Load foreign messages
	for _, f := range pkg.Files {
		if f.Proto.GetPackage() == foreignPackage {
			messages = protobuf.ParseMessagesFromFile(f, f.Proto.GetPackage())
		}
	}
	if len(messages) == 0 {
		return nil, errors.New("could not load foreign messages")
	}

	return messages, nil
}

func getEnumAdditionalSchema(field *protobuf.Field, pkg *protobuf.Protobuf) *Schema {
	var (
		packageName = getPackageName(field.MapValueTypeName())
		enums       []*protobuf.Enum
		schema      = &Schema{
			Type: SchemaTypeString.String(),
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

// ProtoField returns the protobuf field that this schema was generated from.
func (s *Schema) ProtoField() *protobuf.Field {
	return s.field
}
