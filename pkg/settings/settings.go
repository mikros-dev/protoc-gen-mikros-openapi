package settings

import (
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/BurntSushi/toml"
	"github.com/creasty/defaults"
	msettings "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/settings"
)

// Settings contains all settings for the plugin read from the plugin TOML
// file.
type Settings struct {
	Debug                     bool       `toml:"debug" default:"false"`
	AddServiceNameInEndpoints bool       `toml:"add_service_name_in_endpoints" default:"false"`
	Enum                      *Enum      `toml:"enum" default:"{}"`
	Mikros                    *Mikros    `toml:"mikros" default:"{}"`
	Output                    *Output    `toml:"output" default:"{}"`
	Error                     *Error     `toml:"error" default:"{}"`
	Operation                 *Operation `toml:"operation" default:"{}"`

	MikrosSettings *msettings.Settings
}

// Mikros contains all settings related to the protoc-gen-mikros-extensions
// plugin.
type Mikros struct {
	UseOutboundMessages      bool   `toml:"use_outbound_messages" default:"false"`
	UseInboundMessages       bool   `toml:"use_inbound_messages" default:"false"`
	KeepMainModuleFilePrefix bool   `toml:"keep_main_module_file_prefix" default:"false"`
	SettingsFilename         string `toml:"settings_filename"`
}

// Enum contains all settings related to how the plugin handles enums.
type Enum struct {
	RemovePrefix           bool `toml:"remove_prefix" default:"false"`
	RemoveUnspecifiedEntry bool `toml:"remove_unspecified_entry" default:"false"`
}

// Output contains all settings related to the output directory of generated
// OpenAPI files.
type Output struct {
	UseDefaultOut bool   `toml:"use_default_out" default:"false"`
	Path          string `toml:"path" default:"openapi"`
	Filename      string `toml:"filename" default:"openapi.yaml"`
}

// Error contains settings for customizing the default error response.
type Error struct {
	DefaultName        string                `toml:"default_name" default:"DefaultError"`
	DefaultDescription string                `toml:"default_description" default:"The default error response."`
	Fields             map[string]ErrorField `toml:"fields"`
	Responses          []ErrorResponse       `toml:"responses"`
}

// ErrorField defines the basic schema for an error property.
type ErrorField struct {
	Type                 string                `toml:"type"` // string, integer, bool, array, object
	Ref                  string                `toml:"ref"`
	Items                *ErrorField           `toml:"items"`
	Fields               map[string]ErrorField `toml:"fields"`
	AdditionalProperties *ErrorField           `toml:"additional_properties"`
}

// ErrorResponse defines a default error response code that all endpoints
// will have.
type ErrorResponse struct {
	Code        int    `toml:"code"`
	Description string `toml:"description"`
}

// Operation contains settings for customizing behavior of all generated
// endpoints.
type Operation struct {
	DefaultSuccessCode        int    `toml:"default_success_code" default:"200"`
	DefaultSuccessDescription string `toml:"default_success_description" default:"OK"`
}

// LoadSettings loads the settings from the given TOML file.
func LoadSettings(filename string) (*Settings, error) {
	var settings Settings

	if filename != "" {
		file, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		if err := toml.Unmarshal(file, &settings); err != nil {
			return nil, err
		}
	}

	defaultSettings, err := loadDefaultSettings()
	if err != nil {
		return nil, err
	}

	if err := mergo.Merge(&settings, defaultSettings); err != nil {
		return nil, err
	}

	cfg, err := msettings.LoadSettings(settings.Mikros.SettingsFilename)
	if err != nil {
		return nil, fmt.Errorf("could not load mikros plugin settings file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Settings: %w", err)
	}
	settings.MikrosSettings = cfg

	settings.adjustValues()
	return &settings, nil
}

func loadDefaultSettings() (*Settings, error) {
	s := &Settings{}
	if err := defaults.Set(s); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Settings) adjustValues() {
	// Set mikros defaults if no fields are provided
	if len(s.Error.Fields) == 0 {
		s.Error.Fields = map[string]ErrorField{
			"code":         {Type: "integer"},
			"service_name": {Type: "string"},
			"message":      {Type: "string"},
			"destination":  {Type: "string"},
			"kind":         {Type: "string"},
		}
	}

	// Default error codes for all endpoints.
	if len(s.Error.Responses) == 0 {
		s.Error.Responses = []ErrorResponse{
			{Code: 500, Description: "Internal Server Error"},
			{Code: 400, Description: "Bad Request"},
		}
	}
}
