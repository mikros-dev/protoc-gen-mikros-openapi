package lookup

import (
	"fmt"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// FindMainModuleFile returns the main module file for the given protobuf package.
func FindMainModuleFile(pkg *protobuf.Protobuf, cfg *settings.Settings) (*protogen.File, error) {
	mainModuleName := pkg.ModuleName
	if cfg.Mikros.KeepMainModuleFilePrefix {
		mainModuleName = pkg.ModuleName + "_api"
	}

	f, ok := pkg.PackageFiles[mainModuleName]
	if !ok {
		return nil, fmt.Errorf("could not find main module file '%s'", mainModuleName)
	}

	return f, nil
}
