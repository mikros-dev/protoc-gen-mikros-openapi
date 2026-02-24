package metadata

import (
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
)

// Metadata provides optional access to build-time/source information that is
// not part of the OpenAPI document itself.
type Metadata interface {
	// ModuleName is the protobuf module name used during generation.
	ModuleName() string

	// OperationInfo maps a spec operation node back to its HTTP routing info
	// and proto RPC identity. It expects the operation ID to be unique across
	// the whole protobuf package being processed.
	OperationInfo(operationID string) (*OperationInfo, bool)

	// SchemaInfo resolves metadata for the exact schema node instance returned
	// in the spec. It is not stable across copies or serialization.
	SchemaInfo(schema *spec.Schema) (*SchemaInfo, bool)
}

// OperationInfo contains the routing information for a given OpenAPI operation.
type OperationInfo struct {
	Method     string
	Endpoint   string
	InputName  *ProtoName
	OutputName *ProtoName
	Descriptor *descriptorpb.MethodDescriptorProto
}

// ProtoName contains the protobuf type name components.
type ProtoName struct {
	// Raw is the exact protobuf descriptor type name.
	Raw string

	// FullyQualified is Raw without the leading dot.
	FullyQualified string

	// Package is the protobuf package portion.
	Package string

	// Message is the message name without Package.
	Message string
}

// SchemaInfo contains information about a given schema.
type SchemaInfo struct {
	IsRequired        bool
	FieldDescriptor   *descriptorpb.FieldDescriptorProto
}
