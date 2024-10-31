package plugin

import (
	"context"
	"path/filepath"

	"github.com/bufbuild/protoplugin"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/output"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/mikros-dev/protoc-gen-openapi/internal/args"
	mcontext "github.com/mikros-dev/protoc-gen-openapi/internal/context"
)

func Handle(
	_ context.Context,
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

	content, name, err := handleProtogenPlugin(plugin, pluginArgs)
	if err != nil {
		return err
	}

	response := plugin.Response()
	w.AddCodeGeneratorResponseFiles(response.GetFile()...)
	w.SetSupportedFeatures(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) | uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS))
	w.SetFeatureSupportsEditions(descriptorpb.Edition_EDITION_PROTO2, descriptorpb.Edition_EDITION_2024)

	if content != "" {
		w.AddFile(filepath.Join(name, "openapi.yaml"), content)
	}

	return nil
}

func handleProtogenPlugin(plugin *protogen.Plugin, pluginArgs *args.Args) (string, string, error) {
	ctx, err := mcontext.BuildContext(plugin, pluginArgs)
	if err != nil {
		return "", "", err
	}
	if ctx == nil {
		return "", "", nil
	}

	output.Enable(ctx.Settings.Debug)
	output.Println("processing module:", ctx.Openapi.ModuleName())

	content, err := ctx.OutputOpenapi()
	return content, filepath.Join(ctx.Settings.Output.Path, ctx.Openapi.ModuleName()), err
}
