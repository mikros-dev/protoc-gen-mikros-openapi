package lookup

import (
	"slices"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
)

func LoadForeignEnums(enumType string, pkg *protobuf.Protobuf) []*protobuf.Enum {
	var (
		foreignPackage = GetPackageName(enumType)
		enums          []*protobuf.Enum
	)

	// Load foreign enums
	for _, f := range pkg.Files {
		if f.Proto.GetPackage() == foreignPackage {
			enums = protobuf.ParseEnumsFromFile(f)
			break
		}
	}

	return enums
}

func FindEnumByType(enumType string, pkg *protobuf.Protobuf) *protobuf.Enum {
	var enums []*protobuf.Enum
	if GetPackageName(enumType) == pkg.PackageName {
		enums = pkg.Enums
	}
	if GetPackageName(enumType) != pkg.PackageName {
		enums = LoadForeignEnums(enumType, pkg)
	}

	index := slices.IndexFunc(enums, func(enum *protobuf.Enum) bool {
		return enum.Name == TrimPackageName(enumType)
	})
	if index == -1 {
		return nil
	}

	return enums[index]
}
