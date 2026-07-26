package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mcsd/vendors"

	. "mcsd/utils"
)

// InstanceConfig is the JSON schema for config.json in each instance directory.
// It is a pure data definition with no lifecycle methods.
type InstanceConfig struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Vendor     string   `json:"vendor"`
	Version    string   `json:"version"`
	Build      int      `json:"build,omitempty"`
	Binary     string   `json:"binary"`
	JavaArgs   []string `json:"java_args"`
	ServerArgs []string `json:"server_args"`
	Memory     int      `json:"memory"`
}

func (c *InstanceConfig) Validate() error {
	if !vendors.IsValid(c.Vendor) {
		return &ValidationError{Message: fmt.Sprintf("unknown vendor %q", c.Vendor)}
	}
	if c.Memory < 512 {
		return &ValidationError{Message: fmt.Sprintf("memory must be at least 512 MB, got %d", c.Memory)}
	}
	return nil
}

// LoadInstanceConfig reads config.json from an instance directory.
func LoadInstanceConfig(id string) (*InstanceConfig, error) {
	path := filepath.Join(InstanceDir(id), "config.json")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &NotFoundError{Message: fmt.Sprintf("instance %q not found", id)}
		}
		return nil, &ServerError{Message: fmt.Sprintf("open instance config: %s", err.Error())}
	}
	defer f.Close()

	var cfg InstanceConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, &ServerError{Message: fmt.Sprintf("decode instance config: %s", err.Error())}
	}
	if cfg.ID == "" {
		cfg.ID = id
	}
	if cfg.Name == "" {
		cfg.Name = cfg.ID
	}
	return &cfg, nil
}

// WriteInstanceConfig writes config.json to an instance directory atomically.
func WriteInstanceConfig(id string, cfg *InstanceConfig) error {
	dir := InstanceDir(id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ServerError{Message: fmt.Sprintf("create instance dir: %s", err.Error())}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("encode instance config: %s", err.Error())}
	}
	if err := WriteAtomic(filepath.Join(dir, "config.json"), append(data, '\n'), 0644); err != nil {
		return &ServerError{Message: fmt.Sprintf("write instance config: %s", err.Error())}
	}
	return nil
}

// ListInstanceConfigs returns the IDs of all instances that have a config.json.
func ListInstanceConfigs() ([]string, error) {
	entries, err := os.ReadDir(DefaultBasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &ServerError{Message: fmt.Sprintf("read instances dir: %s", err.Error())}
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		configPath := filepath.Join(DefaultBasePath, entry.Name(), "config.json")
		if _, err := os.Stat(configPath); err == nil {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// ValidateIDUnique checks that the given ID is valid and does not already exist.
func ValidateIDUnique(id string) error {
	if err := ValidateID(id); err != nil {
		return &ValidationError{Message: err.Error()}
	}
	if _, err := os.Stat(InstanceDir(id)); err == nil {
		return &ValidationError{Message: fmt.Sprintf("instance %q already exists", id)}
	} else if !os.IsNotExist(err) {
		return &ServerError{Message: fmt.Sprintf("check instance dir: %s", err.Error())}
	}
	return nil
}
