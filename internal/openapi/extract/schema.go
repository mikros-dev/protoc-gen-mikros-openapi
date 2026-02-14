package extract

import (
	"slices"
	"strings"

	"github.com/juliangruber/go-intersect"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

var (
	supportedSchemas = []spec.SchemaType{
		spec.SchemaTypeString,
		spec.SchemaTypeInteger,
		spec.SchemaTypeNumber,
		spec.SchemaTypeBool,
		spec.SchemaTypeObject,
		spec.SchemaTypeArray,
	}
)

// GetAdditionalPropertySchemas returns additional properties schemas for the
// field.
func GetAdditionalPropertySchemas(
	field *protobuf.Field,
	parser *MessageParser,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
) (map[string]*spec.Schema, error) {
	if field.MapValueTypeKind() == protoreflect.MessageKind {
		return getMessageAdditionalSchema(field, parser, methodExtensions, httpCtx)
	}

	if field.MapValueTypeKind() == protoreflect.EnumKind {
		return map[string]*spec.Schema{
			lookup.TrimPackageName(field.MapValueTypeName()): getEnumAdditionalSchema(field, parser.Package),
		}, nil
	}

	return nil, nil
}

func newSchemaFromProtobufField(field *protobuf.Field, pkg *protobuf.Protobuf, cfg *settings.Settings) *spec.Schema {
	var (
		properties = mikros_openapi.LoadFieldExtensions(field.Proto)
		schema     = &spec.Schema{
			Type:  spec.SchemaTypeFromProtobufField(field).String(),
			Field: field, // Saves the field to be used later.
		}
	)

	if properties != nil {
		schema.Required = properties.GetRequired()
		schema.Example = properties.GetExample()
		schema.Description = properties.GetDescription()
		schema.Format = protoFormatToSchemaFormat(properties.GetFormat())
	}

	if field.IsTimestamp() {
		// Timestamps are always formatted as date-time.
		schema.Format = "date-time"
	}

	if field.IsEnum() {
		schema.Enum = getEnumValues(field, pkg, cfg)
	}

	// metadata
	if field.IsProtoStruct() {
		schema.Type = spec.SchemaTypeObject.String()
		schema.AdditionalProperties = &spec.Schema{}
	}

	// interface
	if field.IsProtoValue() {
		schema.Type = spec.SchemaTypeObject.String()
		for _, t := range supportedSchemas {
			schema.AnyOf = append(schema.AnyOf, &spec.Schema{
				Type: t.String(),
			})
		}
	}

	if field.IsMap() {
		// Map should always have keys as string, because JSON does not support
		// other types as keys.
		schema.Type = spec.SchemaTypeObject.String()
		schema.AdditionalProperties = getMapSchema(field)
	}

	if field.IsArray() {
		schema.Type = spec.SchemaTypeArray.String()
		if schema.Items == nil {
			schema.Items = &spec.Schema{
				Type: spec.SchemaTypeFromProtobufField(field).String(),
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

func getMapSchema(field *protobuf.Field) *spec.Schema {
	schema := &spec.Schema{
		Type: schemaTypeFromMapType(field.MapValueTypeKind()).String(),
	}

	if field.MapValueTypeKind() == protoreflect.MessageKind || field.MapValueTypeKind() == protoreflect.EnumKind {
		schema.Type = ""
		schema.Ref = refComponentsSchemas + lookup.TrimPackageName(field.MapValueTypeName())
	}

	return schema
}

func schemaTypeFromMapType(mapType protoreflect.Kind) spec.SchemaType {
	switch mapType {
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return spec.SchemaTypeNumber
	case protoreflect.MessageKind:
		return spec.SchemaTypeObject
	case protoreflect.EnumKind, protoreflect.StringKind, protoreflect.BytesKind:
		return spec.SchemaTypeString
	case protoreflect.BoolKind:
		return spec.SchemaTypeBool
	default:
	}

	return spec.SchemaTypeInteger
}

func getEnumValues(field *protobuf.Field, pkg *protobuf.Protobuf, cfg *settings.Settings) []string {
	var (
		enums       []*protobuf.Enum
		packageName = lookup.GetPackageName(field.TypeName)
		values      []string
	)

	if packageName == pkg.PackageName {
		// Get values from local module enum
		enums = pkg.Enums
	}
	if packageName != pkg.PackageName {
		// Or look for them in foreign packages.
		enums = lookup.LoadForeignEnums(field.TypeName, pkg)
	}

	index := slices.IndexFunc(enums, func(enum *protobuf.Enum) bool {
		return enum.Name == lookup.TrimPackageName(field.TypeName)
	})
	if index != -1 {
		var prefix string
		if cfg.Enum.RemovePrefix {
			prefix = getEnumPrefix(enums[index])
		}

		for _, e := range enums[index].Values {
			if cfg.Enum.RemoveUnspecifiedEntry {
				if strings.HasSuffix(e.ProtoName, "_UNSPECIFIED") {
					continue
				}
			}

			values = append(values, strings.TrimPrefix(e.ProtoName, prefix))
		}
	}

	return values
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

func getMessageAdditionalSchema(
	field *protobuf.Field,
	parser *MessageParser,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
) (map[string]*spec.Schema, error) {
	var (
		packageName = lookup.GetPackageName(field.MapValueTypeName())
		messages    []*protobuf.Message
	)

	if packageName == parser.Package.PackageName {
		messages = parser.Package.Messages
	}
	if packageName != parser.Package.PackageName {
		// find foreign message
		m, err := lookup.LoadForeignMessages(field.MapValueTypeName(), parser.Package)
		if err != nil {
			return nil, err
		}
		messages = m
	}

	// We expect this message to have no internal message fields because
	// we won't dive into them.
	index := slices.IndexFunc(messages, func(msg *protobuf.Message) bool {
		return msg.Name == lookup.TrimPackageName(field.MapValueTypeName())
	})
	if index != -1 {
		return parser.GetMessageSchemas(messages[index], methodExtensions, httpCtx)
	}

	return nil, nil
}

func getEnumAdditionalSchema(field *protobuf.Field, pkg *protobuf.Protobuf) *spec.Schema {
	var (
		packageName = lookup.GetPackageName(field.MapValueTypeName())
		enums       []*protobuf.Enum
		schema      = &spec.Schema{
			Type: spec.SchemaTypeString.String(),
		}
	)

	if packageName == pkg.PackageName {
		// Get values from local module enum
		enums = pkg.Enums
	}
	if packageName != pkg.PackageName {
		// Or look for them in foreign packages.
		enums = lookup.LoadForeignEnums(field.MapValueTypeName(), pkg)
	}

	index := slices.IndexFunc(enums, func(enum *protobuf.Enum) bool {
		return enum.Name == lookup.TrimPackageName(field.MapValueTypeName())
	})
	if index != -1 {
		for _, e := range enums[index].Values {
			schema.Enum = append(schema.Enum, e.ProtoName)
		}
	}

	return schema
}
