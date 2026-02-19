package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

func (p *Parser) buildComponents() (*spec.Components, error) {
	schemas, err := p.collectComponentsSchemas()
	if err != nil {
		return nil, err
	}

	return &spec.Components{
		Schemas:   schemas,
		Responses: p.buildComponentResponses(),
		Security:  buildComponentsSecurity(p.pkg),
	}, nil
}

func (p *Parser) collectComponentsSchemas() (map[string]*spec.Schema, error) {
	schemas := make(map[string]*spec.Schema)

	methodComponents, err := p.collectMethodComponentsSchemas()
	if err != nil {
		return nil, err
	}
	for name, schema := range methodComponents {
		schemas[name] = schema
	}

	for name, schema := range getErrorComponentsSchemas(p.cfg) {
		schemas[name] = schema
	}

	return schemas, nil
}

func (p *Parser) collectMethodComponentsSchemas() (map[string]*spec.Schema, error) {
	var (
		schemas = make(map[string]*spec.Schema)
		parser  = &MessageParser{
			pkg: p.pkg,
			cfg: p.cfg,
		}
	)

	for _, method := range p.pkg.Service.Methods {
		reqMsg, respMsg, err := resolveReqRespMessages(method, p.pkg)
		if err != nil {
			return nil, err
		}

		httpCtx := loadMethodContext(method)

		// Request message schemas are collected only when the method has a body.
		if httpRuleHasBody(httpCtx.httpRule) {
			if err := p.collectRequestSchemas(
				parser,
				reqMsg,
				httpCtx,
				schemas,
			); err != nil {
				return nil, err
			}
		}

		if err := p.collectResponseSchemas(
			parser,
			respMsg,
			httpCtx,
			schemas,
		); err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

// methodContext is a helper structure to hold method-specific context.
type methodContext struct {
	httpRule         *annotations.HttpRule
	pathParameters   []string
	methodExtensions *mikros_extensions.MikrosMethodExtensions
	extensions       *mikros_openapi.OpenapiMethod
}

// loadMethodContext centralizes extraction of annotations and path params.
func loadMethodContext(method *protobuf.Method) *methodContext {
	httpRule := lookup.LoadHTTPRule(method)
	pathParameters, _ := lookup.EndpointInformation(httpRule)

	return &methodContext{
		httpRule:         httpRule,
		pathParameters:   pathParameters,
		methodExtensions: mikros_extensions.LoadMethodExtensions(method.Proto),
		extensions:       mikros_openapi.LoadMethodExtensions(method.Proto),
	}
}

// resolveReqRespMessages finds request/response messages.
func resolveReqRespMessages(
	method *protobuf.Method,
	pkg *protobuf.Protobuf,
) (*protobuf.Message, *protobuf.Message, error) {
	req, err := lookup.FindMessageByName(method.RequestType.Name, pkg)
	if err != nil {
		return nil, nil, err
	}

	resp, err := lookup.FindMessageByName(method.ResponseType.Name, pkg)
	if err != nil {
		return nil, nil, err
	}

	return req, resp, nil
}

func httpRuleHasBody(rule *annotations.HttpRule) bool {
	if rule == nil {
		return false
	}

	return rule.GetBody() != "" || rule.GetPut() != "" || rule.GetPatch() != "" || rule.GetPost() != ""
}

// collectRequestSchemas parses, optionally processes inbound, and merges into accumulator.
func (p *Parser) collectRequestSchemas(
	parser *MessageParser,
	request *protobuf.Message,
	httpCtx *methodContext,
	acc map[string]*spec.Schema,
) error {
	reqSchemas, err := parser.CollectMessageSchemas(request, httpCtx)
	if err != nil {
		return err
	}

	if p.cfg.Mikros.UseInboundMessages {
		if httpCtx.extensions != nil && !httpCtx.extensions.GetDisableInboundProcessing() {
			reqSchemas, err = p.transformSchemasInbound(parser, reqSchemas)
			if err != nil {
				return err
			}
		}
	}

	mergeSchemas(acc, reqSchemas, nil)
	return nil
}

func (p *Parser) transformSchemasInbound(
	parser *MessageParser,
	schemas map[string]*spec.Schema,
) (map[string]*spec.Schema, error) {
	for _, schema := range schemas {
		if len(schema.Properties) == 0 {
			continue
		}

		var (
			properties      = make(map[string]*spec.Schema)
			protoMessage, _ = parser.GetMessageProtobuf(schema)
		)

		for _, property := range schema.Properties {
			protoField := p.resolveProtoField(parser, property)

			inboundName, err := inboundPropertyName(protoField, protoMessage)
			if err != nil {
				return nil, err
			}

			properties[inboundName] = property
		}

		schema.Properties = properties
	}

	return schemas, nil
}

func (p *Parser) resolveProtoField(parser *MessageParser, property *spec.Schema) *protobuf.Field {
	if info, ok := p.getSchemaInfo(property); ok && info.ProtoField != nil {
		return info.ProtoField
	}

	if f, ok := parser.GetFieldProtobuf(property); ok {
		return f
	}

	return nil
}

func inboundPropertyName(protoField *protobuf.Field, protoMessage *protobuf.Message) (string, error) {
	naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
		FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
			ProtoField:   protoField,
			ProtoMessage: protoMessage,
		},
	})
	if err != nil {
		return "", err
	}

	return naming.Inbound(), nil
}

// collectResponseSchemas parses, optionally processes outbound, renames when
// needed, and merges.
func (p *Parser) collectResponseSchemas(
	parser *MessageParser,
	response *protobuf.Message,
	httpCtx *methodContext,
	acc map[string]*spec.Schema,
) error {
	respSchemas, err := parser.CollectMessageSchemas(response, httpCtx)
	if err != nil {
		return err
	}

	var nameConv func(string) string
	if p.cfg.Mikros.UseOutboundMessages {
		converter := mapping.NewMessage(mapping.MessageOptions{
			Settings: p.cfg.MikrosSettings,
		})

		respSchemas, err = p.transformSchemasOutbound(parser, respSchemas, converter)
		if err != nil {
			return err
		}

		nameConv = converter.WireOutputToOutbound
	}

	mergeSchemas(acc, respSchemas, nameConv)
	return nil
}

// mergeSchemas copies all kv pairs from src into dst, optionally renaming keys.
func mergeSchemas(dst, src map[string]*spec.Schema, keyTransform func(string) string) {
	for name, schema := range src {
		if keyTransform != nil {
			name = keyTransform(name)
		}

		dst[name] = schema
	}
}

func (p *Parser) transformSchemasOutbound(
	parser *MessageParser,
	schemas map[string]*spec.Schema,
	converter *mapping.Message,
) (map[string]*spec.Schema, error) {
	for _, schema := range schemas {
		if !schemaNeedsOutboundTransform(schema) {
			continue
		}

		transformSchemaRefOutbound(schema, converter)

		if err := p.transformSchemaPropertiesOutbound(parser, schema, converter); err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

// transformSchemaRefOutbound converts the top-level schema reference if present.
func transformSchemaRefOutbound(schema *spec.Schema, converter *mapping.Message) {
	if schema.Ref == "" {
		return
	}
	schema.Ref = converter.WireOutputToOutbound(schema.Ref)
}

// transformSchemaPropertiesOutbound rebuilds properties map with converted refs and
// outbound JSON tag names.
func (p *Parser) transformSchemaPropertiesOutbound(
	parser *MessageParser,
	schema *spec.Schema,
	converter *mapping.Message,
) error {
	if len(schema.Properties) == 0 {
		return nil
	}

	var (
		protoMessage, _ = parser.GetMessageProtobuf(schema)
		properties      = make(map[string]*spec.Schema)
	)

	for _, property := range schema.Properties {
		var protoField *protobuf.Field
		if info, ok := p.getSchemaInfo(property); ok {
			protoField = info.ProtoField
		}
		if protoField == nil {
			if f, ok := parser.GetFieldProtobuf(property); ok {
				protoField = f
			}
		}

		transformSchemaPropertyRefsOutbound(property, converter)
		naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   protoField,
				ProtoMessage: protoMessage,
			},
		})
		if err != nil {
			return err
		}

		fieldTag, err := mapping.NewFieldTag(&mapping.FieldTagOptions{
			FieldNaming: naming,
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   protoField,
				ProtoMessage: protoMessage,
			},
		})
		if err != nil {
			return err
		}

		properties[fieldTag.OutboundTagFieldName()] = property
	}

	schema.Properties = properties
	return nil
}

// transformSchemaPropertyRefsOutbound converts refs for property, its additionalProperties, and
// items when present.
func transformSchemaPropertyRefsOutbound(property *spec.Schema, converter *mapping.Message) {
	if property.Ref != "" {
		property.Ref = converter.WireOutputToOutbound(property.Ref)
	}

	if property.AdditionalProperties != nil && property.AdditionalProperties.Ref != "" {
		property.AdditionalProperties.Ref = converter.WireOutputToOutbound(property.AdditionalProperties.Ref)
	}

	if property.Items != nil && property.Items.Ref != "" {
		property.Items.Ref = converter.WireOutputToOutbound(property.Items.Ref)
	}
}

func schemaNeedsOutboundTransform(schema *spec.Schema) bool {
	var propertyRef bool
	for _, property := range schema.Properties {
		if property.Ref != "" {
			propertyRef = true
			break
		}

		if property.AdditionalProperties != nil && property.AdditionalProperties.Ref != "" {
			propertyRef = true
			break
		}

		if property.Items != nil && property.Items.Ref != "" {
			propertyRef = true
			break
		}
	}

	return schema.Ref != "" || propertyRef
}

// getErrorComponentsSchemas will return the default error schema and any named
// object schemas referenced from error fields.
func getErrorComponentsSchemas(cfg *settings.Settings) map[string]*spec.Schema {
	var (
		schemas    = make(map[string]*spec.Schema)
		properties = make(map[string]*spec.Schema)
		visiting   = make(map[string]bool)
	)

	for name, field := range cfg.Error.Fields {
		properties[name] = buildErrorSchema(field, schemas, visiting)
	}

	schemas[cfg.Error.DefaultName] = &spec.Schema{
		Type:       schemaTypeObject.String(),
		Properties: properties,
	}

	return schemas
}

func buildErrorSchema(f settings.ErrorField, acc map[string]*spec.Schema, visiting map[string]bool) *spec.Schema {
	if isRefOnlySchema(f) {
		return schemaRef(f.Ref)
	}

	if f.Type == schemaTypeArray.String() {
		return buildErrorArraySchema(f, acc, visiting)
	}

	if f.Type == schemaTypeObject.String() {
		return buildErrorObjectSchema(f, acc, visiting)
	}

	return buildErrorPrimitiveSchema(f)
}

func isRefOnlySchema(f settings.ErrorField) bool {
	return f.Type == "" && f.Ref != ""
}

func buildErrorArraySchema(f settings.ErrorField, acc map[string]*spec.Schema, visiting map[string]bool) *spec.Schema {
	s := &spec.Schema{
		Type: schemaTypeArray.String(),
	}

	if f.Items != nil {
		s.Items = buildErrorSchema(*f.Items, acc, visiting)
		return s
	}

	if f.Ref != "" {
		s.Items = schemaRef(f.Ref)
		return s
	}

	// Emit an array without items rather than panic.
	s.Items = &spec.Schema{}
	return s
}

func buildErrorObjectSchema(f settings.ErrorField, acc map[string]*spec.Schema, visiting map[string]bool) *spec.Schema {
	if isRefOnlyObject(f) {
		return schemaRef(f.Ref)
	}

	if isNamedObjectDefinition(f) {
		emitNamedObjectSchema(f, acc, visiting)
		return schemaRef(f.Ref)
	}

	return buildInlineObjectSchema(f, acc, visiting)
}

func isRefOnlyObject(f settings.ErrorField) bool {
	return f.Ref != "" && len(f.Fields) == 0 && f.AdditionalProperties == nil
}

func isNamedObjectDefinition(f settings.ErrorField) bool {
	return f.Ref != "" && (len(f.Fields) > 0 || f.AdditionalProperties != nil)
}

func emitNamedObjectSchema(f settings.ErrorField, acc map[string]*spec.Schema, visiting map[string]bool) {
	// Prevent infinite recursion on self-references
	if visiting[f.Ref] {
		return
	}

	if _, ok := acc[f.Ref]; ok {
		return
	}

	visiting[f.Ref] = true
	acc[f.Ref] = buildInlineObjectSchema(f, acc, visiting)
	visiting[f.Ref] = false
}

func buildInlineObjectSchema(
	f settings.ErrorField,
	acc map[string]*spec.Schema,
	visiting map[string]bool,
) *spec.Schema {
	s := &spec.Schema{
		Type: schemaTypeObject.String(),
	}

	if len(f.Fields) > 0 {
		s.Properties = buildErrorObjectProperties(f.Fields, acc, visiting)
	}

	if f.AdditionalProperties != nil {
		s.AdditionalProperties = buildErrorSchema(*f.AdditionalProperties, acc, visiting)
	}

	return s
}

func buildErrorObjectProperties(
	fields map[string]settings.ErrorField,
	acc map[string]*spec.Schema,
	visiting map[string]bool,
) map[string]*spec.Schema {
	props := make(map[string]*spec.Schema, len(fields))
	for name, child := range fields {
		props[name] = buildErrorSchema(child, acc, visiting)
	}

	return props
}

func buildErrorPrimitiveSchema(f settings.ErrorField) *spec.Schema {
	if f.Ref != "" {
		return schemaRef(f.Ref)
	}

	return &spec.Schema{
		Type: f.Type,
	}
}

func schemaRef(name string) *spec.Schema {
	return &spec.Schema{
		Ref: refComponentsSchemas + name,
	}
}
