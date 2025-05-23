package settings

import (
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/BurntSushi/toml"
	"github.com/creasty/defaults"
	msettings "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/settings"
)

type Settings struct {
	Debug                     bool    `toml:"debug" default:"false"`
	AddServiceNameInEndpoints bool    `toml:"add_service_name_in_endpoints" default:"false"`
	Enum                      *Enum   `toml:"enum" default:"{}"`
	Mikros                    *Mikros `toml:"mikros" default:"{}"`
	Output                    *Output `toml:"output" default:"{}"`

	MikrosSettings *msettings.Settings
}

type Mikros struct {
	UseOutboundMessages bool   `toml:"use_outbound_messages" default:"false"`
	UseInboundMessages  bool   `toml:"use_inbound_messages" default:"false"`
	SettingsFilename    string `toml:"settings_filename"`
}

type Enum struct {
	RemovePrefix           bool `toml:"remove_prefix" default:"false"`
	RemoveUnspecifiedEntry bool `toml:"remove_unspecified_entry" default:"false"`
}

type Output struct {
	Path string `toml:"path" default:"openapi"`
}

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

	return &settings, nil
}

func loadDefaultSettings() (*Settings, error) {
	s := &Settings{}
	if err := defaults.Set(s); err != nil {
		return nil, err
	}

	return s, nil
}
