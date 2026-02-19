package extract

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/metadata"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
)

func (p *Parser) collectOperationParameters(
	method *protobuf.Method,
	httpRule *annotations.HttpRule,
) ([]*spec.Parameter, error) {
	requestMessage, err := lookup.FindMethodRequestMessage(method, p.pkg)
	if err != nil {
		return nil, err
	}
	if len(requestMessage.Fields) == 0 {
		// No parameters
		return nil, nil
	}

	var (
		params            []*spec.Parameter
		pathParameters, _ = lookup.EndpointInformation(httpRule)
	)

	for _, field := range requestMessage.Fields {
		parameter, info, err := p.buildOperationParameter(method, field, requestMessage, pathParameters, httpRule)
		if err != nil {
			return nil, err
		}

		if parameter.Location == "body" {
			// Body parameters should go with their schema, at the components
			// section.
			continue
		}

		params = append(params, parameter)

		if parameter.Schema != nil {
			// Track parameter schemas for later reference
			p.schemas[parameter.Schema] = &schemaInfo{
				Info:       info,
				ProtoField: field,
			}
		}
	}

	return params, nil
}

func (p *Parser) buildOperationParameter(
	method *protobuf.Method,
	field *protobuf.Field,
	message *protobuf.Message,
	pathParameters []string,
	httpRule *annotations.HttpRule,
) (*spec.Parameter, *metadata.SchemaInfo, error) {
	var (
		properties       = mikros_openapi.LoadFieldExtensions(field.Proto)
		methodExtensions = mikros_extensions.LoadMethodExtensions(method.Proto)
		location         = lookup.FieldLocation(properties, httpRule, methodExtensions, field.Name, pathParameters)
		name             = field.Name
		description      string
	)

	if p.cfg.Mikros.UseInboundMessages {
		naming, err := mapping.NewFieldNaming(&mapping.FieldNamingOptions{
			FieldMappingContextOptions: &mapping.FieldMappingContextOptions{
				ProtoField:   field,
				ProtoMessage: message,
			},
		})
		if err != nil {
			return nil, nil, err
		}

		name = naming.Inbound()
	}

	if properties != nil {
		description = properties.GetDescription()
	}

	return &spec.Parameter{
			Required:    isParameterRequired(properties, location),
			Location:    location,
			Name:        name,
			Description: description,
			Schema:      buildSchemaFromField(field, p.pkg, p.cfg),
		}, &metadata.SchemaInfo{
			FieldDescriptor: field.Proto,
		}, nil
}

func isParameterRequired(properties *mikros_openapi.Property, location string) bool {
	if properties != nil {
		if properties.GetRequired() {
			return true
		}
	}

	return location == "path"
}
