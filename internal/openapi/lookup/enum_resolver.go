package lookup

import "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

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
