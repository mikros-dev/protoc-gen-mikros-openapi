package openapi

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

const (
	refComponentsSchemas = "#/components/schemas/"
)

// Operation describes a single API operation on a path.
type Operation struct {
	Summary         string                `yaml:"summary"`
	Description     string                `yaml:"description"`
	ID              string                `yaml:"operationId"`
	Tags            []string              `yaml:"tags,omitempty"`
	Parameters      []*Parameter          `yaml:"parameters,omitempty"`
	Responses       map[string]*Response  `yaml:"responses,omitempty"`
	RequestBody     *RequestBody          `yaml:"requestBody,omitempty"`
	SecuritySchemes []map[string][]string `yaml:"security,omitempty"`
	ProtobufMethod  *protobuf.Method      `yaml:"-"`

	method   string
	endpoint string
}

func (p *Parser) parsePathItems() (map[string]map[string]*Operation, error) {
	var (
		pathItems = make(map[string]map[string]*Operation)
		converter = mapping.NewMessage(mapping.MessageOptions{
			Settings: p.cfg.MikrosSettings,
		})
	)

	for _, method := range p.pkg.Service.Methods {
		operation, err := p.parseOperation(method, converter)
		if err != nil {
			return nil, err
		}
		if operation == nil {
			continue
		}

		path, ok := pathItems[operation.endpoint]
		if ok {
			path[strings.ToLower(operation.method)] = operation
		}
		if !ok {
			pathItems[operation.endpoint] = map[string]*Operation{
				strings.ToLower(operation.method): operation,
			}
		}
	}

	return pathItems, nil
}

func (p *Parser) parseOperation(
	method *protobuf.Method,
	converter *mapping.Message,
) (*Operation, error) {
	googleAnnotations := mikros_extensions.LoadGoogleAnnotations(method.Proto)
	if googleAnnotations == nil {
		return nil, nil
	}

	endpoint, m := mikros_extensions.GetHTTPEndpoint(googleAnnotations)
	if p.cfg.AddServiceNameInEndpoints {
		endpoint = fmt.Sprintf("/%v%v", strcase.ToKebab(p.pkg.ModuleName), endpoint)
	}

	extensions := mikros_openapi.LoadMethodExtensions(method.Proto)
	if extensions == nil {
		return nil, nil
	}

	parameters, err := p.parseOperationParameters(method, googleAnnotations)
	if err != nil {
		return nil, err
	}

	return &Operation{
		method:          m,
		endpoint:        endpoint,
		Summary:         extensions.GetSummary(),
		Description:     extensions.GetDescription(),
		ID:              method.Name,
		Tags:            extensions.GetTags(),
		Parameters:      parameters,
		Responses:       parseOperationResponses(method, p.cfg, converter),
		RequestBody:     parseRequestBody(method, m, p.pkg),
		SecuritySchemes: parseOperationSecurity(p.pkg),
		ProtobufMethod:  method,
	}, nil
}
