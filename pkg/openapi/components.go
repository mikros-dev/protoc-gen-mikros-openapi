package openapi

import (
	"fmt"
	"slices"

	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// Components describes the components of the API.
type Components struct {
	Schemas   map[string]*Schema   `yaml:"schemas"`
	Responses map[string]*Response `yaml:"responses"`
	Security  map[string]*Security `yaml:"securitySchemes,omitempty"`
}

func parseComponents(pkg *protobuf.Protobuf, cfg *settings.Settings) (*Components, error) {
	schemas, err := parseComponentsSchemas(pkg, cfg)
	if err != nil {
		return nil, err
	}

	return &Components{
		Schemas:   schemas,
		Responses: parseComponentsResponses(pkg, cfg),
		Security:  parseComponentsSecurity(pkg),
	}, nil
}

func parseComponentsSchemas(pkg *protobuf.Protobuf, cfg *settings.Settings) (map[string]*Schema, error) {
	schemas := make(map[string]*Schema)

	methodComponents, err := getMethodComponentsSchemas(pkg, cfg)
	if err != nil {
		return nil, err
	}
	for name, schema := range methodComponents {
		schemas[name] = schema
	}

	for name, schema := range getErrorComponentsSchemas(cfg) {
		schemas[name] = schema
	}

	return schemas, nil
}

type methodHTTPContext struct {
	httpRule       *annotations.HttpRule
	pathParameters []string
}

func getMethodComponentsSchemas(pkg *protobuf.Protobuf, cfg *settings.Settings) (map[string]*Schema, error) {
	var (
		schemas = make(map[string]*Schema)
		parser  = &MessageParser{
			Package:  pkg,
			Settings: cfg,
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
			cfg,
			schemas,
		); err != nil {
			return nil, err
		}

		if err := collectResponseSchemas(
			parser,
			respMsg,
			methodExt,
			httpCtx,
			cfg,
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
	cfg *settings.Settings,
	acc map[string]*Schema,
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

func processInboundMessages(schemas map[string]*Schema) (map[string]*Schema, error) {
	for _, schema := range schemas {
		if len(schema.Properties) == 0 {
			continue
		}

		properties := make(map[string]*Schema)
		for _, property := range schema.Properties {
			naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
				FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
					ProtoField:   property.field,
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
	acc map[string]*Schema,
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
func mergeSchemas(dst, src map[string]*Schema, keyTransform func(string) string) {
	for name, schema := range src {
		if keyTransform != nil {
			name = keyTransform(name)
		}

		dst[name] = schema
	}
}

func processOutboundMessages(schemas map[string]*Schema, cfg *settings.Settings) (map[string]*Schema, error) {
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
func convertSchemaRef(schema *Schema, converter *mapping.Message) {
	if schema.Ref == "" {
		return
	}
	schema.Ref = converter.WireOutputToOutbound(schema.Ref)
}

// renameAndConvertProperties rebuilds properties map with converted refs and
// outbound JSON tag names.
func renameAndConvertProperties(schema *Schema, converter *mapping.Message) error {
	if len(schema.Properties) == 0 {
		return nil
	}

	properties := make(map[string]*Schema)
	for _, property := range schema.Properties {
		convertPropertyRefs(property, converter)
		naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   property.field,
				ProtoMessage: schema.Message,
			},
		})
		if err != nil {
			return err
		}

		fieldTag, err := mapping.NewFieldTag(&mapping.FieldTagOptions{
			FieldNaming: naming,
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   property.field,
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
func convertPropertyRefs(property *Schema, converter *mapping.Message) {
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

func getErrorComponentsSchemas(cfg *settings.Settings) map[string]*Schema {
	properties := make(map[string]*Schema)

	for name, field := range cfg.Error.Fields {
		properties[name] = &Schema{
			Type: field.Type,
		}
	}

	return map[string]*Schema{
		cfg.Error.DefaultName: {
			Type:       SchemaTypeObject.String(),
			Properties: properties,
		},
	}
}

func parseComponentsResponses(pkg *protobuf.Protobuf, cfg *settings.Settings) map[string]*Response {
	responses := make(map[string]*Response)

	for _, method := range pkg.Service.Methods {
		for _, response := range parseMethodComponentsResponses(method, cfg) {
			responses[response.schemaName] = response
		}
	}

	return responses
}

func parseMethodComponentsResponses(method *protobuf.Method, cfg *settings.Settings) []*Response {
	codes := getMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	var responses []*Response
	for _, code := range codes {
		if isSuccessCode(code) {
			continue
		}

		errorName := cfg.Error.DefaultName
		responses = append(responses, &Response{
			schemaName:  errorName,
			Description: "The default error response.",
			Content: map[string]*Media{
				"application/json": {
					Schema: &Schema{
						Ref: refComponentsSchemas + errorName,
					},
				},
			},
		})
	}

	return responses
}
