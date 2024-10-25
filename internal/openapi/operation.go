package openapi

import (
	"strings"

	mextensionspb "github.com/mikros-dev/protoc-gen-mikros-extensions/mikros/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	openapipb "github.com/mikros-dev/protoc-gen-openapi/openapi"
)

const (
	refComponentsSchemas = "#/components/schemas/"
)

type Operation struct {
	Summary         string                `yaml:"summary"`
	Description     string                `yaml:"description"`
	Id              string                `yaml:"operationId"`
	Tags            []string              `yaml:"tags,omitempty"`
	Parameters      []*Parameter          `yaml:"parameters,omitempty"`
	Responses       map[string]*Response  `yaml:"responses,omitempty"`
	RequestBody     *RequestBody          `yaml:"requestBody,omitempty"`
	SecuritySchemes []map[string][]string `yaml:"security,omitempty"`

	method   string
	endpoint string
}

func parsePathItems(pkg *protobuf.Protobuf) (map[string]map[string]*Operation, error) {
	pathItems := make(map[string]map[string]*Operation)

	for _, method := range pkg.Service.Methods {
		operation, err := parseOperation(method, pkg)
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

func parseOperation(method *protobuf.Method, pkg *protobuf.Protobuf) (*Operation, error) {
	googleAnnotations := mextensionspb.LoadGoogleAnnotations(method.Proto)
	if googleAnnotations == nil {
		return nil, nil
	}

	endpoint, m := mextensionspb.GetHttpEndpoint(googleAnnotations)
	extensions := openapipb.LoadMethodExtensions(method.Proto)
	if extensions == nil {
		return nil, nil
	}

	parameters, err := parseOperationParameters(method, googleAnnotations, pkg)
	if err != nil {
		return nil, err
	}

	return &Operation{
		method:          m,
		endpoint:        endpoint,
		Summary:         extensions.GetSummary(),
		Description:     extensions.GetDescription(),
		Id:              method.Name,
		Tags:            extensions.GetTags(),
		Parameters:      parameters,
		Responses:       parseOperationResponses(method),
		RequestBody:     parseRequestBody(method, m, pkg),
		SecuritySchemes: parseOperationSecurity(pkg),
	}, nil
}
