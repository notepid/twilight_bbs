package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the complete BBS configuration.
type Config struct {
	BBS    BBSConfig    `yaml:"bbs"`
	Server ServerConfig `yaml:"server"`
	Paths  PathsConfig  `yaml:"paths"`
	Doors  DoorsConfig  `yaml:"doors"`
}

// BBSConfig holds BBS identity and limits.
type BBSConfig struct {
	Name     string `yaml:"name"`
	Sysop    string `yaml:"sysop"`
	MaxNodes int    `yaml:"max_nodes"`
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

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := &Config{
		BBS: BBSConfig{
			Name:     "Twilight BBS",
			Sysop:    "Sysop",
			MaxNodes: 32,
		},
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
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}
