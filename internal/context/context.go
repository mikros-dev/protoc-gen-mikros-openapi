package context

import (
	"fmt"

	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/args"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/settings"
)

// Context holds the context for the OpenAPI generation. This structure is
// available inside template files.
type Context struct {
	Openapi  *openapi.Openapi
	Settings *settings.Settings
}

// BuildContext builds the main context for the OpenAPI generation.
func BuildContext(plugin *protogen.Plugin, pluginArgs *args.Args) (*Context, error) {
	// Load Mikros-extensions Settings. It returns default values if no
	// file is used.
	cfg, err := settings.LoadSettings(pluginArgs.SettingsFilename)
	if err != nil {
		return nil, fmt.Errorf("could not load Settings file: %w", err)
	}

	// Build the api-specific context
	api, err := openapi.FromProto(plugin, cfg)
	if err != nil {
		return nil, err
	}
	if api == nil {
		// If we're not an HTTP service, we don't need to continue.
		return nil, nil
	}

	return &Context{
		Settings: cfg,
		Openapi:  api,
	}, nil
}

// OutputOpenapi returns the OpenAPI document as a YAML string.
func (c *Context) OutputOpenapi() (string, error) {
	b, err := yaml.Marshal(c.Openapi)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
