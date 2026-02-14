package spec

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
)

// Openapi describes the entire OpenAPI specification.
type Openapi struct {
	Version    string                           `yaml:"openapi"`
	Info       *Info                            `yaml:"info"`
	Servers    []*Server                        `yaml:"servers,omitempty"`
	PathItems  map[string]map[string]*Operation `yaml:"paths,omitempty"`
	Components *Components                      `yaml:"components,omitempty"`

	// private
	ModuleName string
}

// Info describes the service.
type Info struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

// Server describes a server.
type Server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Summary         string                `yaml:"summary"`
	Description     string                `yaml:"description"`
	ID              string                `yaml:"operationId"`
	Tags            []string              `yaml:"tags,omitempty"`
	Parameters      []*Parameter          `yaml:"parameters,omitempty"`
	Responses       map[string]*Response  `yaml:"responses,omitempty"`
	RequestBody     *RequestBody          `yaml:"requestBody,omitempty"`
	SecuritySchemes []map[string][]string `yaml:"security,omitempty"`

	// private
	ProtobufMethod *protobuf.Method `yaml:"-"`
	Method         string
	Endpoint       string
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Required    bool    `yaml:"required"`
	Location    string  `yaml:"in"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty"`
}

// Response describes a single response from an API Operation.
type Response struct {
	Description string            `yaml:"description,omitempty"`
	Content     map[string]*Media `yaml:"content"`

	// private
	SchemaName string
}

// RequestBody describes a request body.
type RequestBody struct {
	Required    bool              `yaml:"required"`
	Description string            `yaml:"description,omitempty"`
	Content     map[string]*Media `yaml:"content"`
}

// Schema represents a swagger schema of a field/parameter/object.
type Schema struct {
	Minimum              int                `yaml:"minimum,omitempty"`
	Maximum              int                `yaml:"maximum,omitempty"`
	Type                 string             `yaml:"type,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	Ref                  string             `yaml:"$ref,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Example              string             `yaml:"example,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`
	Enum                 []string           `yaml:"enum,omitempty"`
	RequiredProperties   []string           `yaml:"required,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty"`
	AnyOf                []*Schema          `yaml:"anyOf,omitempty"`

	// private
	Message    *protobuf.Message `yaml:"-"`
	SchemaType SchemaType
	Required   bool
	Field      *protobuf.Field
}

// IsRequired returns true if the field is required.
func (s *Schema) IsRequired() bool {
	return s.Required
}

// HasAdditionalProperties returns true if the field has additional properties.
func (s *Schema) HasAdditionalProperties() bool {
	return s.AdditionalProperties != nil && s.AdditionalProperties != &Schema{}
}

// ProtoField returns the protobuf field that this schema was generated from.
func (s *Schema) ProtoField() *protobuf.Field {
	return s.Field
}

// Media describes a media type.
type Media struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Components is a structure that describes the components of the API.
type Components struct {
	Schemas   map[string]*Schema   `yaml:"schemas"`
	Responses map[string]*Response `yaml:"responses"`
	Security  map[string]*Security `yaml:"securitySchemes,omitempty"`
}

// Security describes security schemes supported by the API.
type Security struct {
	Type         string `yaml:"type"`
	Scheme       string `yaml:"scheme"`
	BearerFormat string `yaml:"bearerFormat,omitempty"`
}
