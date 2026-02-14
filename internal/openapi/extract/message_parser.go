package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// MessageParser builds OpenAPI schemas from Protobuf messages.
type MessageParser struct {
	Package  *protobuf.Protobuf
	Settings *settings.Settings

	schemas map[string]bool
}

// GetMessageSchemas builds OpenAPI schemas from a Protobuf message.
func (m *MessageParser) GetMessageSchemas(
	message *protobuf.Message,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
) (map[string]*spec.Schema, error) {
	var (
		schemas            = make(map[string]*spec.Schema)
		props              = make(map[string]*spec.Schema)
		requiredProperties []string
	)

	m.addParsedMessage(message.Name)

	for _, f := range message.Fields {
		ext := mikros_openapi.LoadFieldExtensions(f.Proto)
		if m.shouldSkipField(ext) {
			continue
		}

		isRequired, err := m.processField(f, ext, methodExtensions, httpCtx, message, schemas, props)
		if err != nil {
			return nil, err
		}
		if isRequired {
			// Check if the property has a custom name coming from the protobuf
			// annotation.
			fieldName := f.Name
			if ext != nil && ext.GetSchemaName() != "" {
				fieldName = ext.GetSchemaName()
			}

			requiredProperties = append(requiredProperties, fieldName)
		}
	}

	schemas[message.Name] = &spec.Schema{
		Type:               spec.SchemaTypeObject.String(),
		Properties:         props,
		RequiredProperties: requiredProperties,
		Message:            message,
	}

	return schemas, nil
}

func (m *MessageParser) addParsedMessage(name string) {
	if m.schemas == nil {
		m.schemas = make(map[string]bool)
	}

	m.schemas[name] = true
}

func (m *MessageParser) processField(
	field *protobuf.Field,
	ext *mikros_openapi.Property,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	message *protobuf.Message,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	if shouldHandleChildMessage(field) {
		return m.handleChildField(field, methodExtensions, httpCtx, schemas, props)
	}

	if m.shouldSkipNonBodyField(ext, methodExtensions, httpCtx, field.Name, message.ModuleName) {
		return false, nil
	}

	return m.handleRegularField(field, ext, methodExtensions, httpCtx, schemas, props)
}

func (m *MessageParser) shouldSkipField(ext *mikros_openapi.Property) bool {
	return isHidden(ext)
}

func isHidden(ext *mikros_openapi.Property) bool {
	if ext == nil {
		return false
	}

	return ext.GetHideFromSchema()
}

func shouldHandleChildMessage(field *protobuf.Field) bool {
	supportedFieldMessages := field.IsTimestamp() || field.IsProtoStruct() || field.IsProtoAny() || field.IsProtoValue()
	return field.IsMessageFromPackage() || field.IsMessage() && !supportedFieldMessages
}

func (m *MessageParser) handleChildField(
	field *protobuf.Field,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	if err := m.collectChildSchemas(field, methodExtensions, httpCtx, schemas); err != nil {
		return false, err
	}

	ref := m.newRefSchema(field, lookup.TrimPackageName(field.TypeName))
	props[field.Name] = ref

	return ref.IsRequired(), nil
}

func (m *MessageParser) collectChildSchemas(
	field *protobuf.Field,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	schemas map[string]*spec.Schema,
) error {
	if m.isMessageAlreadyParsed(lookup.TrimPackageName(field.TypeName)) {
		return nil
	}

	child, err := m.resolveChildMessage(field)
	if err != nil {
		return err
	}
	if child == nil {
		return nil
	}

	cs, err := m.GetMessageSchemas(child, methodExtensions, httpCtx)
	if err != nil {
		return err
	}

	for n, sc := range cs {
		schemas[n] = sc
	}

	return nil
}

func (m *MessageParser) isMessageAlreadyParsed(name string) bool {
	_, ok := m.schemas[name]
	return ok
}

func (m *MessageParser) resolveChildMessage(field *protobuf.Field) (*protobuf.Message, error) {
	if field.IsMessageFromPackage() {
		return lookup.FindMessageByName(lookup.TrimPackageName(field.TypeName), m.Package)
	}

	if field.IsMessage() {
		return lookup.FindForeignMessage(field.TypeName, m.Package)
	}

	return nil, nil
}

func (m *MessageParser) newRefSchema(
	field *protobuf.Field,
	refDestination string,
) *spec.Schema {
	schema := newSchemaFromProtobufField(field, m.Package, m.Settings)

	if schema.Type == spec.SchemaTypeArray.String() {
		schema.Items = &spec.Schema{
			Ref: refComponentsSchemas + refDestination,
		}
	}

	if schema.Type != spec.SchemaTypeArray.String() {
		schema.Type = "" // Clears the type
		schema.Ref = refComponentsSchemas + refDestination
	}

	return schema
}

func (m *MessageParser) shouldSkipNonBodyField(
	ext *mikros_openapi.Property,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	fieldName, messageModule string,
) bool {
	loc := getFieldLocation(ext, httpCtx.httpRule, methodExtensions, fieldName, httpCtx.pathParameters)
	return loc != "body" && m.Package.ModuleName == messageModule
}

func (m *MessageParser) handleRegularField(
	field *protobuf.Field,
	ext *mikros_openapi.Property,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	name := overrideName(ext, field.Name)
	fs := newSchemaFromProtobufField(field, m.Package, m.Settings)
	props[name] = fs

	if !fs.HasAdditionalProperties() {
		return fs.IsRequired(), nil
	}

	additional, err := GetAdditionalPropertySchemas(field, m, methodExtensions, httpCtx)
	if err != nil {
		return false, err
	}

	for n, sc := range additional {
		schemas[n] = sc
	}

	return fs.IsRequired(), nil
}

func overrideName(ext *mikros_openapi.Property, fallback string) string {
	if ext == nil {
		return fallback
	}

	n := ext.GetSchemaName()
	if n == "" {
		return fallback
	}

	return n
}
