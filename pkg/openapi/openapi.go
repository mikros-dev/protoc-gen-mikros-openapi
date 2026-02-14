package openapi

import (
	"context"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/extract"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// FromProto creates an OpenAPI representation of the given protobuf
// specification. The function loads and translates the main protobuf
// file being handled by the plugin at the moment.
//
// It requires previously loading the mikros-openapi plugin settings so
// it can properly translate to the desired specification.
func FromProto(_ context.Context, plugin *protogen.Plugin, cfg *settings.Settings) (*spec.Openapi, error) {
	pkg, err := protobuf.Parse(protobuf.ParseOptions{
		Plugin: plugin,
	})
	if err != nil {
		return nil, err
	}

	// Only translate protobuf files from HTTP services-like.
	if !isHTTPService(pkg) {
		return nil, nil
	}

	// Create our parser to deal with the protobuf translation into the
	// OpenAPI specification.
	parser := extract.NewParser(pkg, cfg)

	return parser.Do()
}

func isHTTPService(pkg *protobuf.Protobuf) bool {
	return pkg.Service != nil && pkg.Service.IsHTTP()
}
