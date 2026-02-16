package extract

import (
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

type methodHTTPContext struct {
	httpRule       *annotations.HttpRule
	pathParameters []string
}

func (p *Parser) getMethodComponentsSchemas() (map[string]*spec.Schema, error) {
	var (
		schemas = make(map[string]*spec.Schema)
		parser  = &MessageParser{
			Package:  p.pkg,
			Settings: p.cfg,
		}
	)

	for _, method := range p.pkg.Service.Methods {
		httpCtx, methodExt, ext := loadMethodContext(method)

		reqMsg, respMsg, err := resolveReqRespMessages(method, p.pkg)
		if err != nil {
			return nil, err
		}

		// Request message schemas are collected only when the method has a body.
		if httpRuleHasBody(httpCtx.httpRule) {
			if err := collectRequestSchemas(
				parser,
				reqMsg,
				methodExt,
				ext,
				httpCtx,
				p.cfg,
				schemas,
			); err != nil {
				return nil, err
			}
		}

		if err := collectResponseSchemas(
			parser,
			respMsg,
			methodExt,
			httpCtx,
			p.cfg,
			schemas,
		); err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

// loadMethodContext centralizes extraction of annotations and path params.
func loadMethodContext(method *protobuf.Method) (
	*methodHTTPContext,
	*mikros_extensions.MikrosMethodExtensions,
	*mikros_openapi.OpenapiMethod,
) {
	httpRule := lookup.LoadHTTPRule(method)
	methodExtensions := mikros_extensions.LoadMethodExtensions(method.Proto)
	extensions := mikros_openapi.LoadMethodExtensions(method.Proto)
	pathParameters, _ := lookup.EndpointInformation(httpRule)

	return &methodHTTPContext{
		httpRule,
		pathParameters,
	}, methodExtensions, extensions
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
func collectRequestSchemas(
	parser *MessageParser,
	request *protobuf.Message,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	extensions *mikros_openapi.OpenapiMethod,
	httpCtx *methodHTTPContext,
	cfg *settings.Settings,
	acc map[string]*spec.Schema,
) error {
	reqSchemas, err := parser.GetMessageSchemas(request, methodExtensions, httpCtx)
	if err != nil {
		return err
	}
	if cfg.Mikros.UseInboundMessages && !extensions.GetDisableInboundProcessing() {
		reqSchemas, err = processInboundMessages(reqSchemas)
		if err != nil {
			return err
		}
	}

	mergeSchemas(acc, reqSchemas, nil)
	return nil
}

func processInboundMessages(schemas map[string]*spec.Schema) (map[string]*spec.Schema, error) {
	for _, schema := range schemas {
		if len(schema.Properties) == 0 {
			continue
		}

		properties := make(map[string]*spec.Schema)
		for _, property := range schema.Properties {
			naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
				FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
					ProtoField:   property.Field,
					ProtoMessage: schema.Message,
				},
			})
			if err != nil {
				return nil, err
			}

			properties[naming.Inbound()] = property
		}
		schema.Properties = properties
	}

	return schemas, nil
}

// collectResponseSchemas parses, optionally processes outbound, renames when
// needed, and merges.
func collectResponseSchemas(
	parser *MessageParser,
	response *protobuf.Message,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	cfg *settings.Settings,
	acc map[string]*spec.Schema,
) error {
	respSchemas, err := parser.GetMessageSchemas(response, methodExtensions, httpCtx)
	if err != nil {
		return err
	}

	var nameConv func(string) string
	if cfg.Mikros.UseOutboundMessages {
		respSchemas, err = processOutboundMessages(respSchemas, cfg)
		if err != nil {
			return err
		}

		converter := mapping.NewMessage(mapping.MessageOptions{
			Settings: cfg.MikrosSettings,
		})

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

func processOutboundMessages(schemas map[string]*spec.Schema, cfg *settings.Settings) (map[string]*spec.Schema, error) {
	for _, schema := range schemas {
		if !schemaNeedsConversion(schema) {
			continue
		}

		converter := mapping.NewMessage(mapping.MessageOptions{
			Settings: cfg.MikrosSettings,
		})
		convertSchemaRef(schema, converter)

		if err := renameAndConvertProperties(schema, converter); err != nil {
			return nil, err
		}
	}

	return schemas, nil
}

// convertSchemaRef converts the top-level schema reference if present.
func convertSchemaRef(schema *spec.Schema, converter *mapping.Message) {
	if schema.Ref == "" {
		return
	}
	schema.Ref = converter.WireOutputToOutbound(schema.Ref)
}

// renameAndConvertProperties rebuilds properties map with converted refs and
// outbound JSON tag names.
func renameAndConvertProperties(schema *spec.Schema, converter *mapping.Message) error {
	if len(schema.Properties) == 0 {
		return nil
	}

	properties := make(map[string]*spec.Schema)
	for _, property := range schema.Properties {
		convertPropertyRefs(property, converter)
		naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   property.Field,
				ProtoMessage: schema.Message,
			},
		})
		if err != nil {
			return err
		}

		fieldTag, err := mapping.NewFieldTag(&mapping.FieldTagOptions{
			FieldNaming: naming,
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   property.Field,
				ProtoMessage: schema.Message,
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

// convertPropertyRefs converts refs for property, its additionalProperties, and
// items when present.
func convertPropertyRefs(property *spec.Schema, converter *mapping.Message) {
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

func schemaNeedsConversion(schema *spec.Schema) bool {
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
		Type:       spec.SchemaTypeObject.String(),
		Properties: properties,
	}

	return schemas
}

func buildErrorSchema(f settings.ErrorField, acc map[string]*spec.Schema, visiting map[string]bool) *spec.Schema {
	if isRefOnlySchema(f) {
		return schemaRef(f.Ref)
	}

	if f.Type == spec.SchemaTypeArray.String() {
		return buildErrorArraySchema(f, acc, visiting)
	}

	if f.Type == spec.SchemaTypeObject.String() {
		return buildErrorObjectSchema(f, acc, visiting)
	}

	return buildErrorPrimitiveSchema(f)
}

func isRefOnlySchema(f settings.ErrorField) bool {
	return f.Type == "" && f.Ref != ""
}

func buildErrorArraySchema(f settings.ErrorField, acc map[string]*spec.Schema, visiting map[string]bool) *spec.Schema {
	s := &spec.Schema{
		Type: spec.SchemaTypeArray.String(),
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
		Type: spec.SchemaTypeObject.String(),
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

func parseComponentsResponses(pkg *protobuf.Protobuf, cfg *settings.Settings) map[string]*spec.Response {
	responses := make(map[string]*spec.Response)

	for _, method := range pkg.Service.Methods {
		for _, response := range parseMethodComponentsResponses(method, cfg) {
			responses[response.SchemaName] = response
		}
	}

	return responses
}

func parseMethodComponentsResponses(method *protobuf.Method, cfg *settings.Settings) []*spec.Response {
	codes := lookup.LoadMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	var responses []*spec.Response
	for _, code := range codes {
		if lookup.IsSuccessResponseCode(code) {
			continue
		}

		errorName := cfg.Error.DefaultName
		responses = append(responses, &spec.Response{
			SchemaName:  errorName,
			Description: "The default error response.",
			Content: map[string]*spec.Media{
				"application/json": {
					Schema: &spec.Schema{
						Ref: refComponentsSchemas + errorName,
					},
				},
			},
		})
	}

	return responses
}
