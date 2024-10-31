package openapi

import (
	"fmt"
	"slices"

	mextensionspb "github.com/mikros-dev/protoc-gen-mikros-extensions/mikros/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/converters"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-openapi/internal/settings"
)

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

func getMethodComponentsSchemas(pkg *protobuf.Protobuf, settings *settings.Settings) (map[string]*Schema, error) {
	var (
		schemas = make(map[string]*Schema)
		parser  = &MessageParser{
			Package:  pkg,
			Settings: settings,
		}
		converter = converters.NewMessage(converters.MessageOptions{
			Settings: settings.MikrosSettings,
		})
	)

	for _, method := range pkg.Service.Methods {
		var (
			httpRule          = mextensionspb.LoadGoogleAnnotations(method.Proto)
			methodExtensions  = mextensionspb.LoadMethodExtensions(method.Proto)
			pathParameters, _ = getEndpointInformation(httpRule)
		)

		request, err := findMessage(method.RequestType.Name, pkg)
		if err != nil {
			return nil, err
		}

		response, err := findMessage(method.ResponseType.Name, pkg)
		if err != nil {
			return nil, err
		}

		requests, err := parser.GetMessageSchemas(request, httpRule, methodExtensions, pathParameters)
		if err != nil {
			return nil, err
		}
		if settings.Mikros.UseInboundMessages {
			requests = processInboundMessages(requests, settings)
		}
		for name, schema := range requests {
			schemas[name] = schema
		}

		responses, err := parser.GetMessageSchemas(response, httpRule, methodExtensions, pathParameters)
		if err != nil {
			return nil, err
		}
		if settings.Mikros.UseOutboundMessages {
			responses = processOutboundMessages(responses, settings)
		}
		for name, schema := range responses {
			if settings.Mikros.UseOutboundMessages {
				name = converter.WireOutputToOutbound(name)
			}

			schemas[name] = schema
		}
	}

	return schemas, nil
}

func findMessage(msgName string, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	msgIndex := slices.IndexFunc(pkg.Messages, func(msg *protobuf.Message) bool {
		return msg.Name == msgName
	})
	if msgIndex == -1 {
		return nil, fmt.Errorf("could not find message '%s'", msgName)
	}

	return pkg.Messages[msgIndex], nil
}

func processInboundMessages(schemas map[string]*Schema, settings *settings.Settings) map[string]*Schema {
	for _, schema := range schemas {
		if len(schema.Properties) > 0 {
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
	}

	return schemas
}

func processOutboundMessages(schemas map[string]*Schema, settings *settings.Settings) map[string]*Schema {
	for _, schema := range schemas {
		if schemaNeedsConversion(schema) {
			converter := converters.NewMessage(converters.MessageOptions{
				Settings: settings.MikrosSettings,
			})

			if schema.Ref != "" {
				schema.Ref = converter.WireOutputToOutbound(schema.Ref)
			}

			if len(schema.Properties) > 0 {
				properties := make(map[string]*Schema)
				for _, property := range schema.Properties {
					if property.Ref != "" {
						property.Ref = converter.WireOutputToOutbound(property.Ref)
					}

					if property.AdditionalProperties != nil && property.AdditionalProperties.Ref != "" {
						property.AdditionalProperties.Ref = converter.WireOutputToOutbound(property.AdditionalProperties.Ref)
					}

					if property.Items != nil && property.Items.Ref != "" {
						property.Items.Ref = converter.WireOutputToOutbound(property.Items.Ref)
					}

					fieldConverter, _ := converters.NewField(converters.FieldOptions{
						IsHTTPService: true,
						ProtoField:    property.field,
						ProtoMessage:  schema.Message,
					})
					properties[fieldConverter.OutboundJsonTagFieldName()] = property
				}
				schema.Properties = properties
			}
		}
	}

	return schemas
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
			Type: SchemaType_Object.String(),
			Properties: map[string]*Schema{
				"code": {
					Type: SchemaType_Integer.String(),
				},
				"service_name": {
					Type: SchemaType_String.String(),
				},
				"message": {
					Type: SchemaType_String.String(),
				},
				"destination": {
					Type: SchemaType_String.String(),
				},
				"kind": {
					Type: SchemaType_String.String(),
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
