package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// messageParser builds OpenAPI schemas from Protobuf messages.
type messageParser struct {
	pkg            *protobuf.Protobuf
	cfg            *settings.Settings
	parsedMessages map[string]bool                    // keeps track of the messages that have already been parsed
	schemas        map[*spec.Schema]*protobuf.Message // maps created schemas to their Protobuf message
	fields         map[*spec.Schema]*protobuf.Field   // maps created schemas to their Protobuf field
}

// CollectMessageSchemas builds OpenAPI schemas from a Protobuf message.
func (m *messageParser) CollectMessageSchemas(
	message *protobuf.Message,
	methodCtx *methodContext,
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

		isRequired, err := m.processField(f, ext, methodCtx, message, schemas, props)
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
		Type:               schemaTypeObject.String(),
		Properties:         props,
		RequiredProperties: requiredProperties,
	}

	m.trackMessageProtobuf(scm, message)
	schemas[message.Name] = scm

	return schemas, nil
}

func (m *messageParser) trackMessageProtobuf(schema *spec.Schema, message *protobuf.Message) {
	if m.schemas == nil {
		m.schemas = make(map[*spec.Schema]*protobuf.Message)
	}

	m.schemas[schema] = message
}

func (m *messageParser) trackFieldProtobuf(schema *spec.Schema, field *protobuf.Field) {
	if m.fields == nil {
		m.fields = make(map[*spec.Schema]*protobuf.Field)
	}

	m.fields[schema] = field
}

// GetMessageProtobuf returns the Protobuf message associated with a schema.
func (m *messageParser) GetMessageProtobuf(schema *spec.Schema) (*protobuf.Message, bool) {
	p, ok := m.schemas[schema]
	return p, ok
}

// GetFieldProtobuf returns the Protobuf field associated with a schema.
func (m *messageParser) GetFieldProtobuf(schema *spec.Schema) (*protobuf.Field, bool) {
	f, ok := m.fields[schema]
	return f, ok
}

func (m *messageParser) addParsedMessage(name string) {
	if m.parsedMessages == nil {
		m.parsedMessages = make(map[string]bool)
	}

	m.parsedMessages[name] = true
}

func (m *messageParser) processField(
	field *protobuf.Field,
	ext *mikros_openapi.Property,
	methodCtx *methodContext,
	message *protobuf.Message,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	if shouldHandleChildMessage(field) {
		return m.handleChildField(field, methodCtx, schemas, props)
	}

	if m.shouldSkipNonBodyField(ext, methodCtx, field.Name, message.ModuleName) {
		return false, nil
	}

	return m.handleRegularField(field, ext, methodCtx, schemas, props)
}

func (m *messageParser) shouldSkipField(ext *mikros_openapi.Property) bool {
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

func (m *messageParser) handleChildField(
	field *protobuf.Field,
	methodCtx *methodContext,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	if err := m.collectChildSchemas(field, methodCtx, schemas); err != nil {
		return false, err
	}

	ref := m.newRefSchema(field, lookup.TrimPackageName(field.TypeName))
	m.trackFieldProtobuf(ref, field)
	props[field.Name] = ref

	return isFieldRequired(field), nil
}

func (m *messageParser) collectChildSchemas(
	field *protobuf.Field,
	methodCtx *methodContext,
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

	cs, err := m.CollectMessageSchemas(child, methodCtx)
	if err != nil {
		return err
	}

	for n, sc := range cs {
		schemas[n] = sc
	}

	return nil
}

func (m *messageParser) isMessageAlreadyParsed(name string) bool {
	_, ok := m.parsedMessages[name]
	return ok
}

func (m *messageParser) resolveChildMessage(field *protobuf.Field) (*protobuf.Message, error) {
	if field.IsMessageFromPackage() {
		return lookup.FindMessageByName(lookup.TrimPackageName(field.TypeName), m.pkg)
	}

	if field.IsMessage() {
		return lookup.FindForeignMessage(field.TypeName, m.pkg)
	}

	return nil, nil
}

func (m *messageParser) newRefSchema(
	field *protobuf.Field,
	refDestination string,
) *spec.Schema {
	schema := buildSchemaFromField(field, m.pkg, m.cfg)

	if schema.Type == schemaTypeArray.String() {
		schema.Items = &spec.Schema{
			Ref: refComponentsSchemas + refDestination,
		}
	}

	if schema.Type != schemaTypeArray.String() {
		schema.Type = "" // Clears the type
		schema.Ref = refComponentsSchemas + refDestination
	}

	return schema
}

func (m *messageParser) shouldSkipNonBodyField(
	ext *mikros_openapi.Property,
	methodCtx *methodContext,
	fieldName, messageModule string,
) bool {
	if methodCtx == nil || methodCtx.schemaScope != schemaScopeRequest {
		return false
	}

	loc := lookup.FieldLocation(
		ext,
		methodCtx.httpRule,
		methodCtx.methodExtensions,
		fieldName,
		methodCtx.pathParameters,
	)

	return loc != "body" && m.pkg.ModuleName == messageModule
}

func (m *messageParser) handleRegularField(
	field *protobuf.Field,
	ext *mikros_openapi.Property,
	methodCtx *methodContext,
	schemas, props map[string]*spec.Schema,
) (bool, error) {
	name := overrideName(ext, field.Name)
	fs := buildSchemaFromField(field, m.pkg, m.cfg)
	m.trackFieldProtobuf(fs, field)
	props[name] = fs

	if !hasAdditionalProperties(fs) {
		return isFieldRequired(field), nil
	}

	additional, err := collectAdditionalPropertySchemas(field, m, methodCtx)
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
