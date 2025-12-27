package openapi

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/settings"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

// Openapi describes the entire OpenAPI specification.
type Openapi struct {
	Version    string                           `yaml:"openapi"`
	Info       *Info                            `yaml:"info"`
	Servers    []*Server                        `yaml:"servers,omitempty"`
	PathItems  map[string]map[string]*Operation `yaml:"paths,omitempty"`
	Components *Components                      `yaml:"components,omitempty"`

	moduleName string
}

// Info describes the service.
type Info struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

// Server describes a server.
type Server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

// FromProto creates an Openapi instance from the given protoc plugin.
func FromProto(plugin *protogen.Plugin, settings *settings.Settings) (*Openapi, error) {
	pkg, err := protobuf.Parse(protobuf.ParseOptions{
		Plugin: plugin,
	})
	if err != nil {
		return nil, err
	}
	if !isHTTPService(pkg) {
		return nil, nil
	}

	pathItems, err := parsePathItems(pkg, settings)
	if err != nil {
		return nil, err
	}

	components, err := parseComponents(pkg, settings)
	if err != nil {
		return nil, err
	}

	return &Openapi{
		Version:    "3.0.0",
		Info:       parseInfo(pkg),
		Servers:    parseServers(pkg),
		PathItems:  pathItems,
		Components: components,
		moduleName: pkg.ModuleName,
	}, nil
}

func isHTTPService(pkg *protobuf.Protobuf) bool {
	return pkg.Service != nil && pkg.Service.IsHTTP()
}

func parseInfo(pkg *protobuf.Protobuf) *Info {
	var (
		version     = "v0.1.0"
		title       = pkg.ModuleName
		description string
	)

	metadata := mikros_openapi.LoadMetadata(pkg.PackageFiles[pkg.ModuleName+"_api"].Proto)
	if metadata != nil && metadata.GetInfo() != nil {
		title = metadata.GetInfo().GetTitle()
		description = metadata.GetInfo().GetDescription()
		version = metadata.GetInfo().GetVersion()
	}

	return &Info{
		Title:       title,
		Version:     version,
		Description: description,
	}
}

func parseServers(pkg *protobuf.Protobuf) []*Server {
	var (
		metadata = mikros_openapi.LoadMetadata(pkg.PackageFiles[pkg.ModuleName+"_api"].Proto)
		servers  []*Server
	)

	if metadata != nil {
		for _, server := range metadata.GetServer() {
			servers = append(servers, &Server{
				URL:         server.GetUrl(),
				Description: server.GetDescription(),
			})
		}
	}

	return servers
}

// ModuleName returns the name of the module.
func (o *Openapi) ModuleName() string {
	return o.moduleName
}
