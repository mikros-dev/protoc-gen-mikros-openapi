package openapi

import (
	"fmt"
	"slices"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// Parameter describes a single operation parameter.
type Parameter struct {
	Required    bool    `yaml:"required"`
	Location    string  `yaml:"in"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty"`
}

func parseOperationParameters(
	method *protobuf.Method,
	httpRule *annotations.HttpRule,
	pkg *protobuf.Protobuf,
	cfg *settings.Settings,
) ([]*Parameter, error) {
	requestMessage, err := findMethodRequestMessage(method, pkg)
	if err != nil {
		return nil, err
	}
	if len(requestMessage.Fields) == 0 {
		// No parameters
		return nil, nil
	}

	var (
		params            []*Parameter
		pathParameters, _ = getEndpointInformation(httpRule)
	)

	for _, field := range requestMessage.Fields {
		parameter, err := parseOperationParameter(method, field, requestMessage, pathParameters, httpRule, pkg, cfg)
		if err != nil {
			return nil, err
		}

		if parameter.Location == "body" {
			// Body parameters should go with their schema, at the components
			// section.
			continue
		}

		params = append(params, parameter)
	}

	return params, nil
}

func findMethodRequestMessage(method *protobuf.Method, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	msgIndex := slices.IndexFunc(pkg.Messages, func(msg *protobuf.Message) bool {
		return msg.Name == method.RequestType.Name
	})
	if msgIndex == -1 {
		return nil, fmt.Errorf("could not find method request message '%s'", method.RequestType.Name)
	}

	return pkg.Messages[msgIndex], nil
}

func getEndpointInformation(httpRule *annotations.HttpRule) ([]string, string) {
	endpoint, method := mikros_extensions.GetHTTPEndpoint(httpRule)
	return mikros_extensions.RetrieveParameters(endpoint), method
}

func parseOperationParameter(
	method *protobuf.Method,
	field *protobuf.Field,
	message *protobuf.Message,
	pathParameters []string,
	httpRule *annotations.HttpRule,
	pkg *protobuf.Protobuf,
	cfg *settings.Settings,
) (*Parameter, error) {
	var (
		properties       = mikros_openapi.LoadFieldExtensions(field.Proto)
		methodExtensions = mikros_extensions.LoadMethodExtensions(method.Proto)
		location         = getFieldLocation(properties, httpRule, methodExtensions, field.Name, pathParameters)
		name             = field.Name
		description      string
	)

	if cfg.Mikros.UseInboundMessages {
		naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   field,
				ProtoMessage: message,
			},
		})
		if err != nil {
			return nil, err
		}

		name = naming.Inbound()
	}

	if properties != nil {
		description = properties.GetDescription()
	}

	return &Parameter{
		Required:    getParameterMandatory(properties, location),
		Location:    location,
		Name:        name,
		Description: description,
		Schema:      newSchemaFromProtobufField(field, pkg, cfg),
	}, nil
}

func getParameterMandatory(properties *mikros_openapi.Property, location string) bool {
	if properties != nil {
		if properties.GetRequired() {
			return true
		}
	}

	return location == "path"
}
