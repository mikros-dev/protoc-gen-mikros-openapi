package openapi

import (
	"fmt"
	"net/http"
	"slices"

	mextensionspb "github.com/mikros-dev/protoc-gen-mikros-extensions/mikros/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/converters"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-openapi/internal/settings"
	openapipb "github.com/mikros-dev/protoc-gen-openapi/openapi"
)

type Parameter struct {
	Required    bool    `yaml:"required"`
	Location    string  `yaml:"in"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty"`
}

func parseOperationParameters(method *protobuf.Method, httpRule *annotations.HttpRule, pkg *protobuf.Protobuf, settings *settings.Settings) ([]*Parameter, error) {
	requestMessage, err := findMethodRequestMessage(method, pkg)
	if err != nil {
		return nil, err
	}
	if len(requestMessage.Fields) == 0 {
		// No parameters
		return nil, nil
	}

	var (
		params                     []*Parameter
		pathParameters, httpMethod = getEndpointInformation(httpRule)
	)

	if httpMethod == http.MethodPost {
		return nil, nil
	}

	for _, field := range requestMessage.Fields {
		parameter, err := parseOperationParameter(method, field, requestMessage, pathParameters, httpRule, settings)
		if err != nil {
			return nil, err
		}

		if httpMethod == http.MethodPut && parameter.Location == "body" {
			// PUT body parameters should go with its schema, at the components
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
	endpoint, method := mextensionspb.GetHttpEndpoint(httpRule)
	return mextensionspb.RetrieveParameters(endpoint), method
}

func parseOperationParameter(method *protobuf.Method, field *protobuf.Field, message *protobuf.Message, pathParameters []string, httpRule *annotations.HttpRule, settings *settings.Settings) (*Parameter, error) {
	var (
		properties       = openapipb.LoadFieldExtensions(field.Proto)
		methodExtensions = mextensionspb.LoadMethodExtensions(method.Proto)
		location         = getFieldLocation(properties, httpRule, methodExtensions, field.Name, pathParameters)
		name             = field.Name
		description      string
	)

	if settings.Mikros.UseInboundMessages {
		converter, err := converters.NewField(converters.FieldOptions{
			IsHTTPService: true,
			ProtoField:    field,
			ProtoMessage:  message,
		})
		if err != nil {
			return nil, err
		}
		name = converter.InboundName()
	}

	if properties != nil {
		description = properties.GetDescription()
	}

	return &Parameter{
		Required:    getParameterMandatory(properties, location),
		Location:    location,
		Name:        name,
		Description: description,
		Schema:      getParameterSchema(properties, field),
	}, nil
}

func getParameterMandatory(properties *openapipb.Property, location string) bool {
	if properties != nil {
		if properties.GetRequired() {
			return true
		}
	}

	return location == "path"
}

func getParameterSchema(properties *openapipb.Property, field *protobuf.Field) *Schema {
	var (
		example string
		format  string
	)

	if properties != nil {
		example = properties.GetExample()
	}

	if field.IsTimestamp() {
		format = "date-time"
	}

	return &Schema{
		Example: example,
		Format:  format,
		Type:    schemaTypeFromProtobufField(field).String(),
	}
}
