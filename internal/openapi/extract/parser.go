package extract

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	metadata_builder "github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/metadata"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/metadata"
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

	// schemas map all loaded Parameter schemas to their metadata information. It
	// will be populated during the parsing process.
	schemas map[*spec.Schema]*schemaInfo
}

type schemaInfo struct {
	Info       *metadata.SchemaInfo
	ProtoField *protobuf.Field
}

// NewParser creates a new parser for the given protobuf package.
func NewParser(pkg *protobuf.Protobuf, cfg *settings.Settings) *Parser {
	return &Parser{
		pkg:     pkg,
		cfg:     cfg,
		schemas: make(map[*spec.Schema]*schemaInfo),
	}
}

// Parse parses the protobuf file into an OpenAPI specification.
func (p *Parser) Parse() (*spec.Openapi, metadata.Metadata, error) {
	info, err := p.parseInfo()
	if err != nil {
		return nil, nil, err
	}

	pathItems, operationInfo, err := p.parsePathItems()
	if err != nil {
		return nil, nil, err
	}

	components, err := p.parseComponents()
	if err != nil {
		return nil, nil, err
	}

	servers, err := p.parseServers()
	if err != nil {
		return nil, nil, err
	}

	return &spec.Openapi{
			Version:    "3.0.0",
			Info:       info,
			Servers:    servers,
			PathItems:  pathItems,
			Components: components,
		}, metadata_builder.New(metadata_builder.Options{
			ModuleName:    p.pkg.ModuleName,
			OperationInfo: operationInfo,
			SchemaInfo:    p.getMetaSchemaInfo(),
		}), nil
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

	meta := mikros_openapi.LoadMetadata(f.Proto)
	if meta != nil && meta.GetInfo() != nil {
		title = meta.GetInfo().GetTitle()
		description = meta.GetInfo().GetDescription()
		version = meta.GetInfo().GetVersion()
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
		meta    = mikros_openapi.LoadMetadata(f.Proto)
		servers []*spec.Server
	)

	if meta != nil {
		for _, server := range meta.GetServer() {
			servers = append(servers, &spec.Server{
				URL:         server.GetUrl(),
				Description: server.GetDescription(),
			})
		}
	}

	return servers, nil
}

func (p *Parser) parsePathItems() (map[string]map[string]*spec.Operation, map[string]*metadata.OperationInfo, error) {
	var (
		pathItems     = make(map[string]map[string]*spec.Operation)
		operationInfo = make(map[string]*metadata.OperationInfo)
		converter     = mapping.NewMessage(mapping.MessageOptions{
			Settings: p.cfg.MikrosSettings,
		})
	)

	for _, method := range p.pkg.Service.Methods {
		operation, info, err := p.parseOperation(method, converter)
		if err != nil {
			return nil, nil, err
		}
		if operation == nil {
			continue
		}

		path, ok := pathItems[info.Endpoint]
		if ok {
			path[strings.ToLower(info.Method)] = operation
		}
		if !ok {
			pathItems[info.Endpoint] = map[string]*spec.Operation{
				strings.ToLower(info.Method): operation,
			}
		}

		operationInfo[operation.ID] = info
	}

	return pathItems, operationInfo, nil
}

func (p *Parser) parseOperation(
	method *protobuf.Method,
	converter *mapping.Message,
) (*spec.Operation, *metadata.OperationInfo, error) {
	httpRule := lookup.LoadHTTPRule(method)
	if httpRule == nil {
		// The endpoint settings of an RPC are mandatory. It does not make
		// sense to continue if they are not defined.
		return nil, nil, nil
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
		return nil, nil, err
	}

	return &spec.Operation{
			Summary:         summary,
			Description:     description,
			ID:              method.Name,
			Tags:            tags,
			Parameters:      parameters,
			Responses:       p.parseOperationResponses(method, converter),
			RequestBody:     p.parseRequestBody(method, m),
			SecuritySchemes: parseOperationSecurity(p.pkg),
		}, &metadata.OperationInfo{
			Method:     m,
			Endpoint:   endpoint,
			Descriptor: method.Proto,
		}, nil
}

func (p *Parser) getSchemaInfo(schema *spec.Schema) (*schemaInfo, bool) {
	info, ok := p.schemas[schema]
	return info, ok
}

func (p *Parser) getMetaSchemaInfo() map[*spec.Schema]*metadata.SchemaInfo {
	meta := make(map[*spec.Schema]*metadata.SchemaInfo)
	for schema, info := range p.schemas {
		meta[schema] = info.Info
	}

	return meta
}
