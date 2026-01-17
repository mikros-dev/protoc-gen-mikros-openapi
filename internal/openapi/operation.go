package openapi

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/settings"
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

	method   string
	endpoint string
}

func parsePathItems(pkg *protobuf.Protobuf, settings *settings.Settings) (map[string]map[string]*Operation, error) {
	var (
		pathItems = make(map[string]map[string]*Operation)
		converter = mapping.NewMessage(mapping.MessageOptions{
			Settings: settings.MikrosSettings,
		})
	)

	for _, method := range pkg.Service.Methods {
		operation, err := parseOperation(method, pkg, settings, converter)
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

func parseOperation(
	method *protobuf.Method,
	pkg *protobuf.Protobuf,
	settings *settings.Settings,
	converter *mapping.Message,
) (*Operation, error) {
	googleAnnotations := mikros_extensions.LoadGoogleAnnotations(method.Proto)
	if googleAnnotations == nil {
		return nil, nil
	}

	endpoint, m := mikros_extensions.GetHTTPEndpoint(googleAnnotations)
	if settings.AddServiceNameInEndpoints {
		endpoint = fmt.Sprintf("/%v%v", strcase.ToKebab(pkg.ModuleName), endpoint)
	}

	extensions := mikros_openapi.LoadMethodExtensions(method.Proto)
	if extensions == nil {
		return nil, nil
	}

	parameters, err := parseOperationParameters(method, googleAnnotations, pkg, settings)
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
		Responses:       parseOperationResponses(method, settings, converter),
		RequestBody:     parseRequestBody(method, m, pkg),
		SecuritySchemes: parseOperationSecurity(pkg),
	}, nil
}
