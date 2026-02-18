package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// MessageParser builds OpenAPI schemas from Protobuf messages.
type MessageParser struct {
	pkg            *protobuf.Protobuf
	cfg            *settings.Settings
	parsedMessages map[string]bool                    // keeps track of the messages that have already been parsed
	schemas        map[*spec.Schema]*protobuf.Message // maps created schemas to their Protobuf message
	fields         map[*spec.Schema]*protobuf.Field   // maps created schemas to their Protobuf field
}

// GetMessageSchemas builds OpenAPI schemas from a Protobuf message.
func (m *MessageParser) GetMessageSchemas(
	message *protobuf.Message,
	httpCtx *methodContext,
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

		isRequired, err := m.processField(f, ext, httpCtx, message, schemas, props)
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

	scm := &spec.Schema{
		Type:               SchemaTypeObject.String(),
		Properties:         props,
		RequiredProperties: requiredProperties,
	}

	m.trackMessageProtobuf(scm, message)
	schemas[message.Name] = scm

	return schemas, nil
}

func (m *MessageParser) trackMessageProtobuf(schema *spec.Schema, message *protobuf.Message) {
	if m.schemas == nil {
		m.schemas = make(map[*spec.Schema]*protobuf.Message)
	}

	m.schemas[schema] = message
}

func (m *MessageParser) trackFieldProtobuf(schema *spec.Schema, field *protobuf.Field) {
	if m.fields == nil {
		m.fields = make(map[*spec.Schema]*protobuf.Field)
	}

	m.fields[schema] = field
}

// GetMessageProtobuf returns the Protobuf message associated with a schema.
func (m *MessageParser) GetMessageProtobuf(schema *spec.Schema) (*protobuf.Message, bool) {
	p, ok := m.schemas[schema]
	return p, ok
}

// GetFieldProtobuf returns the Protobuf field associated with a schema.
func (m *MessageParser) GetFieldProtobuf(schema *spec.Schema) (*protobuf.Field, bool) {
	f, ok := m.fields[schema]
	return f, ok
}

func (m *MessageParser) addParsedMessage(name string) {
	if m.parsedMessages == nil {
		m.parsedMessages = make(map[string]bool)
	}

	m.parsedMessages[name] = true
}

func (m *MessageParser) processField(
	field *protobuf.Field,
	ext *mikros_openapi.Property,
	httpCtx *methodContext,
	message *protobuf.Message,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	if shouldHandleChildMessage(field) {
		return m.handleChildField(field, httpCtx, schemas, props)
	}

	if m.shouldSkipNonBodyField(ext, httpCtx, field.Name, message.ModuleName) {
		return false, nil
	}

	return m.handleRegularField(field, ext, httpCtx, schemas, props)
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
	httpCtx *methodContext,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	if err := m.collectChildSchemas(field, httpCtx, schemas); err != nil {
		return false, err
	}

	ref := m.newRefSchema(field, lookup.TrimPackageName(field.TypeName))
	m.trackFieldProtobuf(ref, field)
	props[field.Name] = ref

	return isFieldRequired(field), nil
}

func (m *MessageParser) collectChildSchemas(
	field *protobuf.Field,
	httpCtx *methodContext,
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

	cs, err := m.GetMessageSchemas(child, httpCtx)
	if err != nil {
		return err
	}

	for n, sc := range cs {
		schemas[n] = sc
	}

	return nil
}

func (m *MessageParser) isMessageAlreadyParsed(name string) bool {
	_, ok := m.parsedMessages[name]
	return ok
}

func (m *MessageParser) resolveChildMessage(field *protobuf.Field) (*protobuf.Message, error) {
	if field.IsMessageFromPackage() {
		return lookup.FindMessageByName(lookup.TrimPackageName(field.TypeName), m.pkg)
	}

	if field.IsMessage() {
		return lookup.FindForeignMessage(field.TypeName, m.pkg)
	}

	return nil, nil
}

func (m *MessageParser) newRefSchema(
	field *protobuf.Field,
	refDestination string,
) *spec.Schema {
	schema := newSchemaFromProtobufField(field, m.pkg, m.cfg)

	if schema.Type == SchemaTypeArray.String() {
		schema.Items = &spec.Schema{
			Ref: refComponentsSchemas + refDestination,
		}
	}

	if schema.Type != SchemaTypeArray.String() {
		schema.Type = "" // Clears the type
		schema.Ref = refComponentsSchemas + refDestination
	}

	return schema
}

func (m *MessageParser) shouldSkipNonBodyField(
	ext *mikros_openapi.Property,
	httpCtx *methodContext,
	fieldName, messageModule string,
) bool {
	loc := lookup.FieldLocation(ext, httpCtx.httpRule, httpCtx.methodExtensions, fieldName, httpCtx.pathParameters)
	return loc != "body" && m.pkg.ModuleName == messageModule
}

func (m *MessageParser) handleRegularField(
	field *protobuf.Field,
	ext *mikros_openapi.Property,
	httpCtx *methodContext,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	name := overrideName(ext, field.Name)
	fs := newSchemaFromProtobufField(field, m.pkg, m.cfg)
	m.trackFieldProtobuf(fs, field)
	props[name] = fs

	if !fs.HasAdditionalProperties() {
		return isFieldRequired(field), nil
	}

	additional, err := GetAdditionalPropertySchemas(field, m, httpCtx)
	if err != nil {
		return false, err
	}

	for n, sc := range additional {
		schemas[n] = sc
	}

	return isFieldRequired(field), nil
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
