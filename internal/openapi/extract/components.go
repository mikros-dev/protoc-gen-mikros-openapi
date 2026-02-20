package extract

import (
	"strings"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/genproto/googleapis/api/annotations"

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
		parser  = &messageParser{
			pkg: p.pkg,
			cfg: p.cfg,
		}
	)

	for _, method := range p.pkg.Service.Methods {
		methodCtx := p.buildMethodContext(method)
		if err := p.loadMethodMessages(methodCtx); err != nil {
			return nil, err
		}

		// Request message schemas are collected only when the method has a body.
		if httpRuleHasBody(methodCtx.httpRule) {
			if err := p.collectRequestSchemas(
				parser,
				methodCtx,
				schemas,
			); err != nil {
				return nil, err
			}
		}

		if err := p.collectResponseSchemas(
			parser,
			methodCtx,
			schemas,
		); err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

func httpRuleHasBody(rule *annotations.HttpRule) bool {
	if rule == nil {
		return false
	}

	return rule.GetBody() != "" || rule.GetPut() != "" || rule.GetPatch() != "" || rule.GetPost() != ""
}

// collectRequestSchemas parses, optionally processes inbound, and merges into accumulator.
func (p *Parser) collectRequestSchemas(
	parser *messageParser,
	methodCtx *methodContext,
	acc map[string]*spec.Schema,
) error {
	reqSchemas, err := parser.CollectMessageSchemas(methodCtx.requestMessage, methodCtx)
	if err != nil {
		return err
	}

	if p.cfg.Mikros.UseInboundMessages {
		if methodCtx.extensions != nil && !methodCtx.extensions.GetDisableInboundProcessing() {
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
	parser *messageParser,
	schemas map[string]*spec.Schema,
) (map[string]*spec.Schema, error) {
	for _, schema := range schemas {
		err := transformSchema(schema, transformRules{
			TransformPropertyName: func(parent *spec.Schema, name string, property *spec.Schema) (string, error) {
				protoMessage, ok := parser.GetMessageProtobuf(parent)
				if !ok {
					return name, nil
				}

				protoField := p.resolveProtoField(parser, property)
				return inboundPropertyName(protoField, protoMessage)
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

func (p *Parser) resolveProtoField(parser *messageParser, property *spec.Schema) *protobuf.Field {
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
	parser *messageParser,
	methodCtx *methodContext,
	acc map[string]*spec.Schema,
) error {
	respSchemas, err := parser.CollectMessageSchemas(methodCtx.responseMessage, methodCtx)
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
	parser *messageParser,
	schemas map[string]*spec.Schema,
	converter *mapping.Message,
) (map[string]*spec.Schema, error) {
	transformRef := func(ref string) string {
		if strings.HasPrefix(ref, refComponentsSchemas) {
			name := strings.TrimPrefix(ref, refComponentsSchemas)
			return refComponentsSchemas + converter.WireOutputToOutbound(name)
		}

		// With an unknown ref shape we don't risk corrupting it
		return ref
	}

	for _, schema := range schemas {
		err := transformSchema(schema, transformRules{
			TransformRef: transformRef,
			TransformPropertyName: func(parent *spec.Schema, name string, property *spec.Schema) (string, error) {
				protoMessage, ok := parser.GetMessageProtobuf(parent)
				if !ok {
					return name, nil
				}

				protoField := p.resolveProtoField(parser, property)
				return outboundPropertyName(protoField, protoMessage)
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

func outboundPropertyName(protoField *protobuf.Field, protoMessage *protobuf.Message) (string, error) {
	naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
		FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
			ProtoField:   protoField,
			ProtoMessage: protoMessage,
		},
	})
	if err != nil {
		return "", err
	}

	fieldTag, err := mapping.NewFieldTag(&mapping.FieldTagOptions{
		FieldNaming: naming,
		FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
			ProtoField:   protoField,
			ProtoMessage: protoMessage,
		},
	})
	if err != nil {
		return "", err
	}

	return fieldTag.OutboundTagFieldName(), nil
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
