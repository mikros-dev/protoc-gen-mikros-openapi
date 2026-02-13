package plugin

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/protoplugin"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/ctxutil"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/log"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/args"
	pcontext "github.com/mikros-dev/protoc-gen-mikros-openapi/internal/context"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// Handle is the entry point for the plugin to be processed by protoc/buf.
func Handle(
	ctx context.Context,
	_ protoplugin.PluginEnv,
	w protoplugin.ResponseWriter,
	r protoplugin.Request,
) error {
	pluginArgs, err := args.NewArgsFromString(r.Parameter())
	if err != nil {
		return err
	}

	plugin, err := protogen.Options{}.New(r.CodeGeneratorRequest())
	if err != nil {
		return err
	}
	cfg, err := settings.LoadSettings(pluginArgs.SettingsFilename)
	if err != nil {
		return fmt.Errorf("could not load settings file: %w", err)
	}

	logger := log.New(log.LoggerOptions{
		Verbose: cfg.Debug,
		Prefix:  "[mikros-openapi]",
	})
	ctx = ctxutil.WithLogger(ctx, logger)

	content, name, err := handleProtogenPlugin(ctx, plugin, cfg)
	if err != nil {
		return err
	}

	response := plugin.Response()
	w.AddCodeGeneratorResponseFiles(response.GetFile()...)
	w.SetFeatureSupportsEditions(descriptorpb.Edition_EDITION_PROTO2, descriptorpb.Edition_EDITION_2024)
	w.SetSupportedFeatures(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) |
		uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS))

	if content != "" {
		w.AddFile(name, content)
		logger.Println("generated file:", name)
	}

	return nil
}

func handleProtogenPlugin(ctx context.Context, plugin *protogen.Plugin, cfg *settings.Settings) (string, string, error) {
	logger := ctxutil.LoggerFromContext(ctx)

	// Build the context for the template generation
	tplContext, err := pcontext.BuildContext(ctx, plugin, cfg)
	if err != nil {
		return "", "", err
	}
	if tplContext == nil {
		return "", "", nil
	}

	logger.Println("processing module:", tplContext.Openapi.ModuleName())
	content, err := tplContext.OutputOpenapi()

	// Defines the destination directory for the generated file
	outputDir := filepath.Join(tplContext.Settings.Output.Path, tplContext.Openapi.ModuleName())
	if cfg.Output.UseDefaultOut {
		outputDir = ""
	}

	// And the filename
	filename := cfg.Output.Filename
	if filename == "" {
		filename = "openapi.yaml"
	}

	return content, filepath.Join(outputDir, filename), err
}
