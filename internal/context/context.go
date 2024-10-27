package context

import (
	"fmt"

	"github.com/goccy/go-yaml"
	mcontext "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/context"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/mikros-dev/protoc-gen-openapi/internal/args"
	"github.com/mikros-dev/protoc-gen-openapi/internal/openapi"
	"github.com/mikros-dev/protoc-gen-openapi/internal/settings"
)

type Context struct {
	Openapi  *openapi.Openapi
	Settings *settings.Settings
	Mikros   *mcontext.Context
}

func BuildContext(plugin *protogen.Plugin, pluginArgs *args.Args) (*Context, error) {
	// Load Mikros-extensions Settings. It returns default values if no
	// file is used.
	cfg, err := settings.LoadSettings(pluginArgs.SettingsFilename)
	if err != nil {
		return nil, fmt.Errorf("could not load Settings file: %w", err)
	}

	msettings, err := cfg.MikrosSettings()
	if err != nil {
		return nil, err
	}

	// Build Mikros-extensions context to have some data properly loaded.
	ctx, err := mcontext.BuildContext(mcontext.BuildContextOptions{
		PluginName: pluginArgs.GetPluginName(),
		Settings:   msettings,
		Plugin:     plugin,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build templates context: %w", err)
	}
	if !ctx.IsHTTPService() {
		// If we're not an HTTP service, we don't need to continue.
		return nil, nil
	}

	// Build the api specific context
	api, err := openapi.FromProto(plugin, ctx)
	if err != nil {
		return nil, err
	}

	return &Context{
		Mikros:   ctx,
		Settings: cfg,
		Openapi:  api,
	}, nil
}

func (c *Context) OutputOpenapi() (string, error) {
	b, err := yaml.Marshal(c.Openapi)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
