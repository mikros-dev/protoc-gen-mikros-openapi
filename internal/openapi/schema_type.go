package openapi

import (
	descriptor "google.golang.org/protobuf/types/descriptorpb"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
)

type SchemaType int

const (
	SchemaType_Unspecified SchemaType = iota
	SchemaType_Object
	SchemaType_String
	SchemaType_Array
	SchemaType_Bool
	SchemaType_Integer
	SchemaType_Number
)

func schemaTypeFromProtobufField(field *protobuf.Field) SchemaType {
	switch field.Type {
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return SchemaType_String

	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return SchemaType_String

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return SchemaType_Bool

	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return SchemaType_Number

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if field.IsTimestamp() {
			return SchemaType_String
		}

	default:
		return SchemaType_Integer
	}

	return SchemaType_Unspecified
}

func (s SchemaType) String() string {
	switch s {
	case SchemaType_Integer:
		return "integer"

	case SchemaType_Number:
		return "number"

	case SchemaType_Bool:
		return "boolean"

	case SchemaType_Object:
		return "object"

	case SchemaType_String:
		return "string"

	case SchemaType_Array:
		return "array"

	default:
	}

	return "unspecified"
}
