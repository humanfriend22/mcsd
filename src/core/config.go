package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	. "mcsd/utils"
)

type Config struct {
	MemoryBudget int `json:"memory_budget"`
	Port         int `json:"port"` // daemon HTTP port; 0 = use default 8080
}

func LoadConfig() (*Config, error) {
	file, err := os.Open(GlobalConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &NotFoundError{Message: fmt.Sprintf("config not found at %s — run 'mcsd init' to set up this host", GlobalConfigPath)}
		}
		return nil, &ServerError{Message: fmt.Sprintf("open global config: %s", err.Error())}
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, &ServerError{Message: fmt.Sprintf("decode global config: %s", err.Error())}
	}

	total, err := TotalSystemMemory()
	if err != nil {
		return nil, &ServerError{Message: fmt.Sprintf("fetch total system memory: %s", err.Error())}
	}
	if config.MemoryBudget > total {
		// Cap in memory only — don't write back. Budget was set on a machine with more RAM.
		config.MemoryBudget = total - 512
	}

	return &config, nil
}

func WriteConfig(config *Config) error {
	if err := os.MkdirAll(filepath.Dir(GlobalConfigPath), 0755); err != nil {
		return &ServerError{Message: fmt.Sprintf("create config dir: %s", err.Error())}
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("encode config: %s", err.Error())}
	}
	return WriteAtomic(GlobalConfigPath, append(data, '\n'), 0644)
}
