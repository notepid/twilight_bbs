package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the BBS configuration (excluding BBS identity settings which are in the database).
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Paths    PathsConfig    `yaml:"paths"`
	Doors    DoorsConfig    `yaml:"doors"`
	Transfer TransferConfig `yaml:"transfer"`
}

// ServerConfig holds network listener settings.
type ServerConfig struct {
	TelnetPort int `yaml:"telnet_port"`
	SSHPort    int `yaml:"ssh_port"`
	HealthPort int `yaml:"health_port"`
}

// PathsConfig holds filesystem paths for assets and data.
type PathsConfig struct {
	Menus    string `yaml:"menus"`
	Text     string `yaml:"text"`
	Doors    string `yaml:"doors"`
	Data     string `yaml:"data"`
	Database string `yaml:"database"`
}

// DoorsConfig holds DOS door integration settings.
type DoorsConfig struct {
	DosemuPath string `yaml:"dosemu_path"`
	DriveC     string `yaml:"drive_c"`
}

// TransferConfig holds file transfer protocol settings.
type TransferConfig struct {
	SexyzPath string `yaml:"sexyz_path"`
}

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := &Config{
		Server: ServerConfig{
			TelnetPort: 2323,
			SSHPort:    2222,
			HealthPort: 2223,
		},
		Paths: PathsConfig{
			Menus:    "./assets/menus",
			Text:     "./assets/text",
			Doors:    "./assets/doors",
			Data:     "./data",
			Database: "./data/twilight.db",
		},
		Doors: DoorsConfig{
			DosemuPath: "/usr/bin/dosemu",
			DriveC:     "./doors/drive_c",
		},
		Transfer: TransferConfig{
			SexyzPath: "/usr/local/bin/sexyz",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}
