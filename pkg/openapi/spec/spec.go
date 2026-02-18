package spec

// Openapi describes the OpenAPI specification.
type Openapi struct {
	Version    string                           `yaml:"openapi"`
	Info       *Info                            `yaml:"info"`
	Servers    []*Server                        `yaml:"servers,omitempty"`
	PathItems  map[string]map[string]*Operation `yaml:"paths,omitempty"`
	Components *Components                      `yaml:"components,omitempty"`
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
}

// RequestBody describes a request body.
type RequestBody struct {
	Required    bool              `yaml:"required"`
	Description string            `yaml:"description,omitempty"`
	Content     map[string]*Media `yaml:"content"`
}

// Media describes a media type.
type Media struct {
	Schema *Schema `json:"schema,omitempty"`
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
}

// HasAdditionalProperties returns true if the field has additional properties.
func (s *Schema) HasAdditionalProperties() bool {
	return s.AdditionalProperties != nil && s.AdditionalProperties != &Schema{}
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
