package openapi

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// parseEnums parses all enums declared inside all protobuf files, even the
// ones declared inside "imported" protobuf files. It will map their values
// using the type fully-qualified name as key.
func parseEnums(plugin *protogen.Plugin) map[string][]string {
	enums := make(map[string][]string)

	for _, file := range plugin.Files {
		for name, values := range loadEnumsFromFile(file) {
			enums[name] = values
		}
	}

	return enums
}

func loadEnumsFromFile(file *protogen.File) map[string][]string {
	enums := make(map[string][]string)

	for _, enum := range file.Enums {
		desc := enum.Desc.(protoreflect.Descriptor)
		values := loadEnumFromProtogenEnum(enum)
		enums[string(desc.FullName())] = values
	}

	return enums
}

func loadEnumFromProtogenEnum(enum *protogen.Enum) []string {
	var values []string
	for _, v := range enum.Values {
		values = append(values, trimEnumPrefix(v.GoIdent.GoName))
	}

	return values
}

func trimEnumPrefix(value string) string {
	prefix := getEnumPrefix(value)
	return strings.TrimPrefix(value, fmt.Sprintf("%s_%s_", prefix, strcase.ToScreamingSnake(prefix)))
}

func getEnumPrefix(value string) string {
	parts := strings.Split(value, "_")
	return parts[0]
}
