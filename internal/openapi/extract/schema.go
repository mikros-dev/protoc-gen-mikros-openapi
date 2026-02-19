package extract

import (
	"slices"
	"strings"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

var (
	supportedSchemas = []schemaType{
		schemaTypeString,
		schemaTypeInteger,
		schemaTypeNumber,
		schemaTypeBool,
		schemaTypeObject,
		schemaTypeArray,
	}
)

// collectAdditionalPropertySchemas returns additional properties schemas for the
// field.
func collectAdditionalPropertySchemas(
	field *protobuf.Field,
	parser *messageParser,
	methodCtx *methodContext,
) (map[string]*spec.Schema, error) {
	if field.MapValueTypeKind() == protoreflect.MessageKind {
		return getMessageAdditionalSchema(field, parser, methodCtx)
	}

	if field.MapValueTypeKind() == protoreflect.EnumKind {
		return map[string]*spec.Schema{
			lookup.TrimPackageName(field.MapValueTypeName()): getEnumAdditionalSchema(field, parser.pkg),
		}, nil
	}

	return nil, nil
}

func buildSchemaFromField(field *protobuf.Field, pkg *protobuf.Protobuf, cfg *settings.Settings) *spec.Schema {
	schema := buildBaseSchema(field)

	applyProtobufSpecialCases(schema, field, pkg, cfg)
	applyFieldExtensionOverrides(schema, field)
	applyContainerShape(schema, field)
	normalizeSchemaInvariants(schema, field)

	return schema
}

func buildBaseSchema(field *protobuf.Field) *spec.Schema {
	return &spec.Schema{
		Type: schemaTypeFromProtobufField(field).String(),
	}
}

func applyProtobufSpecialCases(
	schema *spec.Schema,
	field *protobuf.Field,
	pkg *protobuf.Protobuf,
	cfg *settings.Settings,
) {
	if field.IsTimestamp() {
		// Timestamps are always formatted as date-time.
		schema.Format = "date-time"
	}

	if field.IsEnum() {
		schema.Enum = getEnumValues(field, pkg, cfg)
	}

	if field.IsProtoStruct() {
		schema.Type = schemaTypeObject.String()
		schema.AdditionalProperties = &spec.Schema{}
	}

	if field.IsProtoValue() {
		schema.Type = schemaTypeObject.String()
		for _, t := range supportedSchemas {
			schema.AnyOf = append(schema.AnyOf, &spec.Schema{
				Type: t.String(),
			})
		}
	}
}

func applyFieldExtensionOverrides(schema *spec.Schema, field *protobuf.Field) {
	properties := mikros_openapi.LoadFieldExtensions(field.Proto)
	if properties == nil {
		return
	}

	schema.Example = properties.GetExample()
	schema.Description = properties.GetDescription()

	format := protoFormatToSchemaFormat(properties.GetFormat())
	if format == "" {
		return
	}

	schema.Format = format
}

func applyContainerShape(schema *spec.Schema, field *protobuf.Field) {
	if field.IsMap() {
		// Map should always have keys as string, because JSON does not support
		// other types as keys.
		schema.Type = schemaTypeObject.String()
		schema.AdditionalProperties = getMapSchema(field)
	}

	if !field.IsArray() {
		return
	}

	schema.Type = schemaTypeArray.String()
}

func normalizeSchemaInvariants(schema *spec.Schema, field *protobuf.Field) {
	if schema.Type != schemaTypeArray.String() {
		return
	}

	if schema.Items != nil {
		return
	}

	schema.Items = &spec.Schema{
		Type: schemaTypeFromProtobufField(field).String(),
	}
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

func schemaTypeFromMapType(mapType protoreflect.Kind) schemaType {
	switch mapType {
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return schemaTypeNumber
	case protoreflect.MessageKind:
		return schemaTypeObject
	case protoreflect.EnumKind, protoreflect.StringKind, protoreflect.BytesKind:
		return schemaTypeString
	case protoreflect.BoolKind:
		return schemaTypeBool
	default:
	}

	return schemaTypeInteger
}

func getEnumValues(field *protobuf.Field, pkg *protobuf.Protobuf, cfg *settings.Settings) []string {
	var (
		values []string
	)

	enum := lookup.FindEnumByType(field.TypeName, pkg)
	if enum != nil {
		var prefix string
		if cfg.Enum.RemovePrefix {
			prefix = getEnumPrefix(enum)
		}

		for _, e := range enum.Values {
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
	var (
		p1 = strings.Split(s1, "_")
		p2 = strings.Split(s2, "_")
	)

	limit := len(p1)
	if len(p2) < limit {
		limit = len(p2)
	}

	var prefix []string
	for i := 0; i < limit; i++ {
		if p1[i] != p2[i] {
			break
		}
		prefix = append(prefix, p1[i])
	}
	if len(prefix) == 0 {
		return ""
	}

	return strings.Join(prefix, "_") + "_"
}

func getMessageAdditionalSchema(
	field *protobuf.Field,
	parser *messageParser,
	methodCtx *methodContext,
) (map[string]*spec.Schema, error) {
	var (
		packageName = lookup.GetPackageName(field.MapValueTypeName())
		messages    []*protobuf.Message
	)

	if packageName == parser.pkg.PackageName {
		messages = parser.pkg.Messages
	}
	if packageName != parser.pkg.PackageName {
		// find foreign message
		m, err := lookup.LoadForeignMessages(field.MapValueTypeName(), parser.pkg)
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
		return parser.CollectMessageSchemas(messages[index], methodCtx)
	}

	return nil, nil
}

func getEnumAdditionalSchema(field *protobuf.Field, pkg *protobuf.Protobuf) *spec.Schema {
	schema := &spec.Schema{
		Type: schemaTypeString.String(),
	}

	enum := lookup.FindEnumByType(field.MapValueTypeName(), pkg)
	if enum == nil {
		return schema
	}

	for _, e := range enum.Values {
		schema.Enum = append(schema.Enum, e.ProtoName)
	}

	return schema
}

func isFieldRequired(field *protobuf.Field) bool {
	properties := mikros_openapi.LoadFieldExtensions(field.Proto)
	if properties == nil {
		return false
	}

	return properties.GetRequired()
}

func hasAdditionalProperties(schema *spec.Schema) bool {
	if schema == nil {
		return false
	}

	return schema.AdditionalProperties != nil
}
