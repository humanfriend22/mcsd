package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "mcsd/helpers"
)

type Config struct {
	MemoryBudget int `json:"memory_budget"`
	Port         int `json:"port,omitempty"` // daemon HTTP port; 0 = use default 8080
}

func TotalSystemMemory() (int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "MemTotal:") {
			fields := strings.Fields(scanner.Text()) // ["MemTotal:", "16327584", "kB"]
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			return int(kb / 1024), nil // kB -> MiB
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

func LoadConfig() (*Config, error) {
	file, err := os.Open(GlobalConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s — run 'mcsd init' to set up this host", GlobalConfigPath)
		}
		return nil, fmt.Errorf("open global config: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode global config: %w", err)
	}

	total, err := TotalSystemMemory()
	if err != nil {
		return nil, fmt.Errorf("fetch total system memory: %w", err)
	}
	if config.MemoryBudget > total {
		// Cap in memory only — don't write back. Budget was set on a machine with more RAM.
		config.MemoryBudget = total - 512
	}

	return &config, nil
}

func WriteConfig(config *Config) error {
	if err := os.MkdirAll(filepath.Dir(GlobalConfigPath), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return WriteAtomic(GlobalConfigPath, append(data, '\n'), 0644)
}
