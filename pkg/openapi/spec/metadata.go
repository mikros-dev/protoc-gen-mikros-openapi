package spec

import (
	"google.golang.org/protobuf/types/descriptorpb"
)

// Metadata provides optional access to build-time/source information that is
// not part of the OpenAPI document itself.
type Metadata interface {
	// ModuleName is the protobuf module name used during generation.
	ModuleName() string

	// OperationInfo maps a spec operation node back to its HTTP routing info.
	// and proto RPC identity.
	OperationInfo(operationID string) (*OperationInfo, bool)

	// SchemaInfo maps a spec schema node back to its proto message descriptor.
	SchemaInfo(schema *Schema) (*SchemaInfo, bool)
}

// OperationInfo contains the routing information for a given OpenAPI operation.
type OperationInfo struct {
	Method     string
	Endpoint   string
	Descriptor *descriptorpb.MethodDescriptorProto
}

// SchemaInfo contains information about a given schema.
type SchemaInfo struct {
	FieldDescriptor   *descriptorpb.FieldDescriptorProto
	MessageDescriptor *descriptorpb.DescriptorProto
}
