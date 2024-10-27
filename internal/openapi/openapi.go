package openapi

import (
	mcontext "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/context"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/mikros-dev/protoc-gen-openapi/internal/settings"
	openapipb "github.com/mikros-dev/protoc-gen-openapi/openapi"
)

type Openapi struct {
	Version    string                           `yaml:"openapi"`
	Info       *Info                            `yaml:"info"`
	Servers    []*Server                        `yaml:"servers,omitempty"`
	PathItems  map[string]map[string]*Operation `yaml:"paths,omitempty"`
	Components *Components                      `yaml:"components,omitempty"`
}

type Info struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

type Server struct {
	Url         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

func FromProto(plugin *protogen.Plugin, ctx *mcontext.Context, settings *settings.Settings) (*Openapi, error) {
	pkg, err := protobuf.Parse(protobuf.ParseOptions{
		Plugin: plugin,
	})
	if err != nil {
		return nil, err
	}

	pathItems, err := parsePathItems(pkg)
	if err != nil {
		return nil, err
	}

	components, err := parseComponents(pkg, settings)
	if err != nil {
		return nil, err
	}

	return &Openapi{
		Version:    "3.0.0",
		Info:       parseInfo(ctx, pkg),
		Servers:    parseServers(ctx, pkg),
		PathItems:  pathItems,
		Components: components,
	}, nil
}

func parseInfo(ctx *mcontext.Context, pkg *protobuf.Protobuf) *Info {
	var (
		version     = "v0.1.0"
		title       = ctx.ModuleName
		description string
	)

	if metadata := openapipb.LoadMetadata(pkg.PackageFiles[ctx.ModuleName+"_api"].Proto); metadata != nil && metadata.GetInfo() != nil {
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

func parseServers(ctx *mcontext.Context, pkg *protobuf.Protobuf) []*Server {
	var (
		metadata = openapipb.LoadMetadata(pkg.PackageFiles[ctx.ModuleName+"_api"].Proto)
		servers  []*Server
	)

	if metadata != nil {
		for _, server := range metadata.GetServer() {
			servers = append(servers, &Server{
				Url:         server.GetUrl(),
				Description: server.GetDescription(),
			})
		}
	}

	return servers
}
