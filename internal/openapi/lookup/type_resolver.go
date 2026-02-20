package lookup

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
)

// GetPackageName returns the package name of the given message type.
func GetPackageName(msgType string) string {
	parts := strings.Split(strings.TrimPrefix(msgType, "."), ".")
	return strings.Join(parts[:len(parts)-1], ".")
}

// TrimPackageName removes the package name from the given message type.
func TrimPackageName(name string) string {
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

// FindMessageByName returns the message with the given name from the given package.
func FindMessageByName(msgName string, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	msgIndex := slices.IndexFunc(pkg.Messages, func(msg *protobuf.Message) bool {
		return msg.Name == msgName
	})
	if msgIndex == -1 {
		return nil, fmt.Errorf("could not find message '%s'", msgName)
	}

	return pkg.Messages[msgIndex], nil
}

// FindMethodRequestMessage returns the request message of the given method.
func FindMethodRequestMessage(method *protobuf.Method, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	return FindMessageByName(method.RequestType.Name, pkg)
}

// FindForeignMessage returns the message with the given name from a foreign
// package. A foreign package is a package that is not part of the current
// protobuf package being processed.
func FindForeignMessage(msgType string, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	messages, err := LoadForeignMessages(msgType, pkg)
	if err != nil {
		return nil, err
	}

	msgIndex := slices.IndexFunc(messages, func(msg *protobuf.Message) bool {
		return msg.Name == TrimPackageName(msgType)
	})
	if msgIndex == -1 {
		return nil, fmt.Errorf("could not find foreign message '%s'", msgType)
	}

	return messages[msgIndex], nil
}

// LoadForeignMessages loads all messages from a foreign package. A foreign
// package is a package that is not part of the current protobuf package being
// processed.
func LoadForeignMessages(msgType string, pkg *protobuf.Protobuf) ([]*protobuf.Message, error) {
	var (
		foreignPackage = GetPackageName(msgType)
		messages       []*protobuf.Message
	)

	for _, f := range pkg.Files {
		if f.Proto.GetPackage() != foreignPackage {
			continue
		}

		messages = protobuf.ParseMessagesFromFile(f, f.Proto.GetPackage())
	}
	if len(messages) == 0 {
		return nil, errors.New("could not load foreign messages")
	}

	return messages, nil
}

// LoadForeignEnums loads all enums from a foreign package. A foreign package
// is a package that is not part of the current protobuf package being processed.
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

// FindEnumByType returns the enum with the given name from the given package.
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
