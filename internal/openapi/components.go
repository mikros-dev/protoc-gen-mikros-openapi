package openapi

import (
	"fmt"
	"slices"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/converters"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mikros_extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/settings"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

// Components describes the components of the API.
type Components struct {
	Schemas   map[string]*Schema   `yaml:"schemas"`
	Responses map[string]*Response `yaml:"responses"`
	Security  map[string]*Security `yaml:"securitySchemes,omitempty"`
}

func parseComponents(pkg *protobuf.Protobuf, settings *settings.Settings) (*Components, error) {
	schemas, err := parseComponentsSchemas(pkg, settings)
	if err != nil {
		return nil, err
	}

	return &Components{
		Schemas:   schemas,
		Responses: parseComponentsResponses(pkg),
		Security:  parseComponentsSecurity(pkg),
	}, nil
}

func parseComponentsSchemas(pkg *protobuf.Protobuf, settings *settings.Settings) (map[string]*Schema, error) {
	schemas := make(map[string]*Schema)

	methodComponents, err := getMethodComponentsSchemas(pkg, settings)
	if err != nil {
		return nil, err
	}
	for name, schema := range methodComponents {
		schemas[name] = schema
	}

	for name, schema := range getErrorComponentsSchemas() {
		schemas[name] = schema
	}

	return schemas, nil
}

type methodHTTPContext struct {
	httpRule       *annotations.HttpRule
	pathParameters []string
}

func getMethodComponentsSchemas(pkg *protobuf.Protobuf, settings *settings.Settings) (map[string]*Schema, error) {
	var (
		schemas = make(map[string]*Schema)
		parser  = &MessageParser{
			Package:  pkg,
			Settings: settings,
		}
	)

	for _, method := range pkg.Service.Methods {
		httpCtx, methodExt, ext := loadMethodContext(method)

		reqMsg, respMsg, err := resolveReqRespMessages(method, pkg)
		if err != nil {
			return nil, err
		}

		if err := collectRequestSchemas(
			parser,
			reqMsg,
			methodExt,
			ext,
			httpCtx,
			settings,
			schemas,
		); err != nil {
			return nil, err
		}

		if err := collectResponseSchemas(
			parser,
			respMsg,
			methodExt,
			httpCtx,
			settings,
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
	httpRule := mikros_extensions.LoadGoogleAnnotations(method.Proto)
	methodExtensions := mikros_extensions.LoadMethodExtensions(method.Proto)
	extensions := mikros_openapi.LoadMethodExtensions(method.Proto)
	pathParameters, _ := getEndpointInformation(httpRule)

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
	req, err := FindMessageByName(method.RequestType.Name, pkg)
	if err != nil {
		return nil, nil, err
	}

	resp, err := FindMessageByName(method.ResponseType.Name, pkg)
	if err != nil {
		return nil, nil, err
	}

	return req, resp, nil
}

// FindMessageByName finds a message by its name inside the loaded protobuf
// messages.
func FindMessageByName(msgName string, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	msgIndex := slices.IndexFunc(pkg.Messages, func(msg *protobuf.Message) bool {
		return msg.Name == msgName
	})
	if msgIndex == -1 {
		return nil, fmt.Errorf("could not find message '%s'", msgName)
	}

	return pkg.Messages[msgIndex], nil
}

// collectRequestSchemas parses, optionally processes inbound, and merges into accumulator.
func collectRequestSchemas(
	parser *MessageParser,
	request *protobuf.Message,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	extensions *mikros_openapi.OpenapiMethod,
	httpCtx *methodHTTPContext,
	settings *settings.Settings,
	acc map[string]*Schema,
) error {
	reqSchemas, err := parser.GetMessageSchemas(request, methodExtensions, httpCtx)
	if err != nil {
		return err
	}
	if settings.Mikros.UseInboundMessages && !extensions.GetDisableInboundProcessing() {
		reqSchemas = processInboundMessages(reqSchemas, settings)
	}

	mergeSchemas(acc, reqSchemas, nil)
	return nil
}

func processInboundMessages(schemas map[string]*Schema, settings *settings.Settings) map[string]*Schema {
	for _, schema := range schemas {
		if len(schema.Properties) == 0 {
			continue
		}

		properties := make(map[string]*Schema)
		for _, property := range schema.Properties {
			converter, _ := converters.NewField(converters.FieldOptions{
				IsHTTPService: true,
				ProtoField:    property.field,
				ProtoMessage:  schema.Message,
				Settings:      settings.MikrosSettings,
			})

			properties[converter.InboundName()] = property
		}
		schema.Properties = properties
	}

	return schemas
}

// collectResponseSchemas parses, optionally processes outbound, renames when
// needed, and merges.
func collectResponseSchemas(
	parser *MessageParser,
	response *protobuf.Message,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	httpCtx *methodHTTPContext,
	settings *settings.Settings,
	acc map[string]*Schema,
) error {
	respSchemas, err := parser.GetMessageSchemas(response, methodExtensions, httpCtx)
	if err != nil {
		return err
	}

	var nameConv func(string) string
	if settings.Mikros.UseOutboundMessages {
		respSchemas = processOutboundMessages(respSchemas, settings)
		converter := converters.NewMessage(converters.MessageOptions{
			Settings: settings.MikrosSettings,
		})

		nameConv = converter.WireOutputToOutbound
	}

	mergeSchemas(acc, respSchemas, nameConv)
	return nil
}

// mergeSchemas copies all kv pairs from src into dst, optionally renaming keys.
func mergeSchemas(dst, src map[string]*Schema, keyTransform func(string) string) {
	for name, schema := range src {
		if keyTransform != nil {
			name = keyTransform(name)
		}

		dst[name] = schema
	}
}

func processOutboundMessages(schemas map[string]*Schema, settings *settings.Settings) map[string]*Schema {
	for _, schema := range schemas {
		if !schemaNeedsConversion(schema) {
			continue
		}

		converter := converters.NewMessage(converters.MessageOptions{
			Settings: settings.MikrosSettings,
		})

		convertSchemaRef(schema, converter)
		renameAndConvertProperties(schema, converter)
	}

	return schemas
}

// convertSchemaRef converts the top-level schema reference if present.
func convertSchemaRef(schema *Schema, converter *converters.Message) {
	if schema.Ref == "" {
		return
	}
	schema.Ref = converter.WireOutputToOutbound(schema.Ref)
}

// renameAndConvertProperties rebuilds properties map with converted refs and
// outbound JSON tag names.
func renameAndConvertProperties(schema *Schema, converter *converters.Message) {
	if len(schema.Properties) == 0 {
		return
	}

	properties := make(map[string]*Schema)
	for _, property := range schema.Properties {
		convertPropertyRefs(property, converter)

		fieldConverter, _ := converters.NewField(converters.FieldOptions{
			IsHTTPService: true,
			ProtoField:    property.field,
			ProtoMessage:  schema.Message,
		})

		properties[fieldConverter.OutboundJsonTagFieldName()] = property
	}

	schema.Properties = properties
}

// convertPropertyRefs converts refs for property, its additionalProperties, and
// items when present.
func convertPropertyRefs(property *Schema, converter *converters.Message) {
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

func schemaNeedsConversion(schema *Schema) bool {
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

func getErrorComponentsSchemas() map[string]*Schema {
	return map[string]*Schema{
		"DefaultError": {
			Type: SchemaTypeObject.String(),
			Properties: map[string]*Schema{
				"code": {
					Type: SchemaTypeInteger.String(),
				},
				"service_name": {
					Type: SchemaTypeString.String(),
				},
				"message": {
					Type: SchemaTypeString.String(),
				},
				"destination": {
					Type: SchemaTypeString.String(),
				},
				"kind": {
					Type: SchemaTypeString.String(),
				},
			},
		},
	}
}

func parseComponentsResponses(pkg *protobuf.Protobuf) map[string]*Response {
	responses := make(map[string]*Response)

	for _, method := range pkg.Service.Methods {
		for _, response := range parseMethodComponentsResponses(method) {
			responses[response.schemaName] = response
		}
	}

	return responses
}

func parseMethodComponentsResponses(method *protobuf.Method) []*Response {
	codes := getMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	var responses []*Response
	for _, code := range codes {
		if isSuccessCode(code) {
			continue
		}

		responses = append(responses, &Response{
			schemaName:  "DefaultError",
			Description: "The default error response.",
			Content: map[string]*Media{
				"application/json": {
					Schema: &Schema{
						Ref: refComponentsSchemas + "DefaultError",
					},
				},
			},
		})
	}

	return responses
}
