package spec

// This should be private

import (
	descriptor "google.golang.org/protobuf/types/descriptorpb"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
)

// SchemaType describes the type of the schema.
type SchemaType int

// Supported schema types.
const (
	SchemaTypeUnspecified SchemaType = iota
	SchemaTypeObject
	SchemaTypeString
	SchemaTypeArray
	SchemaTypeBool
	SchemaTypeInteger
	SchemaTypeNumber
)

func SchemaTypeFromProtobufField(field *protobuf.Field) SchemaType {
	switch field.Type {
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return SchemaTypeString
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return SchemaTypeString
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return SchemaTypeBool
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return SchemaTypeNumber
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if field.IsTimestamp() {
			return SchemaTypeString
		}
	default:
		return SchemaTypeInteger
	}

	return SchemaTypeUnspecified
}

func (s SchemaType) String() string {
	switch s {
	case SchemaTypeInteger:
		return "integer"
	case SchemaTypeNumber:
		return "number"
	case SchemaTypeBool:
		return "boolean"
	case SchemaTypeObject:
		return "object"
	case SchemaTypeString:
		return "string"
	case SchemaTypeArray:
		return "array"
	default:
	}

	return "unspecified"
}
