package extract

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

// schemaType describes the type of the schema.
type schemaType int

// Supported schema types.
const (
	schemaTypeUnspecified schemaType = iota
	schemaTypeObject
	schemaTypeString
	schemaTypeArray
	schemaTypeBool
	schemaTypeInteger
	schemaTypeNumber
)

// schemaTypeFromProtobufField returns the schema type for the given protobuf field.
func schemaTypeFromProtobufField(field *protobuf.Field) schemaType {
	switch field.Type {
	case descriptor.FieldDescriptorProto_TYPE_STRING, descriptor.FieldDescriptorProto_TYPE_BYTES:
		return schemaTypeString
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return schemaTypeString
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return schemaTypeBool
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return schemaTypeNumber
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if field.IsTimestamp() {
			return schemaTypeString
		}

		// Messages will always default to object types.
		return schemaTypeObject
	default:
		return schemaTypeInteger
	}
}

func (s schemaType) String() string {
	switch s {
	case schemaTypeInteger:
		return "integer"
	case schemaTypeNumber:
		return "number"
	case schemaTypeBool:
		return "boolean"
	case schemaTypeObject:
		return "object"
	case schemaTypeString:
		return "string"
	case schemaTypeArray:
		return "array"
	default:
	}

	return "unspecified"
}
