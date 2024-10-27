package openapi

import (
	mextensionspb "github.com/mikros-dev/protoc-gen-mikros-extensions/mikros/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-openapi/internal/settings"
)

type Components struct {
	Schemas   map[string]*Schema   `yaml:"schemas"`
	Responses map[string]*Response `yaml:"responses"`
	Security  map[string]*Security `yaml:"securitySchemes"`
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
	schemas := make(map[string]*Schema)
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

		requests, err := getMessageSchemas(request, pkg, httpRule, methodExtensions, pathParameters, settings)
		if err != nil {
			return nil, err
		}
		for name, schema := range requests {
			schemas[name] = schema
		}

		responses, err := getMessageSchemas(response, pkg, httpRule, methodExtensions, pathParameters, settings)
		if err != nil {
			return nil, err
		}
		for name, schema := range responses {
			schemas[name] = schema
		}
	}

	return schemas, nil
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
