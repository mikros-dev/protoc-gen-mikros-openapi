package extract

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

const (
	refComponentsSchemas = "#/components/schemas/"
)

// Parser is the internal parser mechanism for translating a protobuf file
// into an OpenAPI specification.
type Parser struct {
	pkg *protobuf.Protobuf
	cfg *settings.Settings
}

func NewParser(pkg *protobuf.Protobuf, cfg *settings.Settings) *Parser {
	return &Parser{
		pkg: pkg,
		cfg: cfg,
	}
}

func (p *Parser) Do() (*spec.Openapi, error) {
	info, err := p.parseInfo()
	if err != nil {
		return nil, err
	}

	pathItems, err := p.parsePathItems()
	if err != nil {
		return nil, err
	}

	components, err := p.parseComponents()
	if err != nil {
		return nil, err
	}

	servers, err := p.parseServers()
	if err != nil {
		return nil, err
	}

	return &spec.Openapi{
		Version:    "3.0.0",
		Info:       info,
		Servers:    servers,
		PathItems:  pathItems,
		Components: components,
		ModuleName: p.pkg.ModuleName,
	}, nil
}

func (p *Parser) parseInfo() (*spec.Info, error) {
	f, err := lookup.FindMainModuleFile(p.pkg, p.cfg)
	if err != nil {
		return nil, err
	}

	var (
		version     = "v0.1.0"
		title       = p.pkg.ModuleName
		description string
	)

	metadata := mikros_openapi.LoadMetadata(f.Proto)
	if metadata != nil && metadata.GetInfo() != nil {
		title = metadata.GetInfo().GetTitle()
		description = metadata.GetInfo().GetDescription()
		version = metadata.GetInfo().GetVersion()
	}

	return &spec.Info{
		Title:       title,
		Version:     version,
		Description: description,
	}, nil
}

func (p *Parser) parseServers() ([]*spec.Server, error) {
	f, err := lookup.FindMainModuleFile(p.pkg, p.cfg)
	if err != nil {
		return nil, err
	}

	var (
		metadata = mikros_openapi.LoadMetadata(f.Proto)
		servers  []*spec.Server
	)

	if metadata != nil {
		for _, server := range metadata.GetServer() {
			servers = append(servers, &spec.Server{
				URL:         server.GetUrl(),
				Description: server.GetDescription(),
			})
		}
	}

	return servers, nil
}

func (p *Parser) parsePathItems() (map[string]map[string]*spec.Operation, error) {
	var (
		pathItems = make(map[string]map[string]*spec.Operation)
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

		path, ok := pathItems[operation.Endpoint]
		if ok {
			path[strings.ToLower(operation.Method)] = operation
		}
		if !ok {
			pathItems[operation.Endpoint] = map[string]*spec.Operation{
				strings.ToLower(operation.Method): operation,
			}
		}
	}

	return pathItems, nil
}

func (p *Parser) parseOperation(method *protobuf.Method, converter *mapping.Message) (*spec.Operation, error) {
	httpRule := lookup.LoadHTTPRule(method)
	if httpRule == nil {
		// The endpoint settings of an RPC are mandatory. It does not make
		// sense to continue if they are not defined.
		return nil, nil
	}

	endpoint, m := lookup.HTTPEndpoint(httpRule)
	if p.cfg.AddServiceNameInEndpoints {
		endpoint = fmt.Sprintf("/%v%v", strcase.ToKebab(p.pkg.ModuleName), endpoint)
	}

	var (
		summary     = method.Name
		description = ""
		tags        = []string{
			p.pkg.ModuleName,
		}
	)

	extensions := mikros_openapi.LoadMethodExtensions(method.Proto)
	if extensions != nil {
		if extensions.GetSummary() != "" {
			summary = extensions.GetSummary()
		}
		if len(extensions.GetTags()) > 0 {
			tags = extensions.GetTags()
		}
		description = extensions.GetDescription()
	}

	parameters, err := p.parseOperationParameters(method, httpRule)
	if err != nil {
		return nil, err
	}

	return &spec.Operation{
		Method:          m,
		Endpoint:        endpoint,
		Summary:         summary,
		Description:     description,
		ID:              method.Name,
		Tags:            tags,
		Parameters:      parameters,
		Responses:       parseOperationResponses(method, p.cfg, converter),
		RequestBody:     parseRequestBody(method, m, p.pkg),
		SecuritySchemes: parseOperationSecurity(p.pkg),
		ProtobufMethod:  method,
	}, nil
}

func (p *Parser) parseComponents() (*spec.Components, error) {
	schemas, err := p.parseComponentsSchemas()
	if err != nil {
		return nil, err
	}

	return &spec.Components{
		Schemas:   schemas,
		Responses: parseComponentsResponses(p.pkg, p.cfg),
		Security:  parseComponentsSecurity(p.pkg),
	}, nil
}

func (p *Parser) parseComponentsSchemas() (map[string]*spec.Schema, error) {
	schemas := make(map[string]*spec.Schema)

	methodComponents, err := p.getMethodComponentsSchemas()
	if err != nil {
		return nil, err
	}
	for name, schema := range methodComponents {
		schemas[name] = schema
	}

	for name, schema := range getErrorComponentsSchemas(p.cfg) {
		schemas[name] = schema
	}

	return schemas, nil
}
