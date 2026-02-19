package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/metadata"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
)

func (p *Parser) collectOperationParameters(methodCtx *methodContext) ([]*spec.Parameter, error) {
	requestMessage := methodCtx.requestMessage
	if len(requestMessage.Fields) == 0 {
		// No parameters
		return nil, nil
	}

	var (
		params []*spec.Parameter
	)

	for _, field := range requestMessage.Fields {
		parameter, info, err := p.buildOperationParameter(methodCtx, field, requestMessage)
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
	methodCtx *methodContext,
	field *protobuf.Field,
	message *protobuf.Message,
) (*spec.Parameter, *metadata.SchemaInfo, error) {
	var (
		properties = mikros_openapi.LoadFieldExtensions(field.Proto)
		location   = lookup.FieldLocation(
			properties,
			methodCtx.httpRule,
			methodCtx.methodExtensions,
			field.Name,
			methodCtx.pathParameters,
		)
		name        = field.Name
		description string
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
