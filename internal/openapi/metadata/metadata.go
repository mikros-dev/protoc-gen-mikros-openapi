package metadata

import (
	"strings"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/metadata"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
)

// Metadata holds the metadata for the OpenAPI specification.
type Metadata struct {
	moduleName    string
	operationInfo map[string]*metadata.OperationInfo
	schemaInfo    map[*spec.Schema]*metadata.SchemaInfo
}

// Options holds the options for the Metadata instance.
type Options struct {
	ModuleName    string
	OperationInfo map[string]*metadata.OperationInfo
	SchemaInfo    map[*spec.Schema]*metadata.SchemaInfo
}

// New creates a new Metadata instance.
func New(options Options) *Metadata {
	return &Metadata{
		moduleName:    options.ModuleName,
		operationInfo: options.OperationInfo,
		schemaInfo:    options.SchemaInfo,
	}
}

// ModuleName returns the module name.
func (m *Metadata) ModuleName() string {
	return m.moduleName
}

// OperationInfo returns the operation info for the given operation ID.
func (m *Metadata) OperationInfo(operationID string) (*metadata.OperationInfo, bool) {
	info, ok := m.operationInfo[operationID]
	return info, ok
}

// SchemaInfo returns the schema info for the given schema.
func (m *Metadata) SchemaInfo(schema *spec.Schema) (*metadata.SchemaInfo, bool) {
	info, ok := m.schemaInfo[schema]
	return info, ok
}

// NewProtoName creates a metadata.ProtoName based on the type name passed.
func NewProtoName(typeName string) *metadata.ProtoName {
	var (
		raw = typeName
		fq = strings.TrimPrefix(typeName, ".")
		pkg = ""
		msg = fq
	)

	if i := strings.LastIndexByte(fq, '.'); i >= 0 {
		pkg = fq[:i]
		msg = fq[i+1:]
	}

	return &metadata.ProtoName{
		Raw:            raw,
		FullyQualified: fq,
		Package:        pkg,
		Message:        msg,
	}
}