package core

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "mcsd/utils"
)

const DefaultPreferredMemory = 6144

func TotalSystemMemory() (int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "MemTotal:") {
			fields := strings.Fields(scanner.Text())
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			return int(kb / 1024), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, &ServerError{Message: "MemTotal not found in /proc/meminfo"}
}

func TotalReservedMemory(excludeID string) (int, error) {
	units, err := SDManager.List("mcsd-instance@*.service")
	if err != nil {
		return 0, &ServerError{Message: fmt.Sprintf("list servers: %s", err.Error())}
	}
	var total int
	for _, unit := range units {
		if unit.State != "active" {
			continue
		}
		id := UnitToID(unit.Name)
		if id == excludeID {
			continue
		}
		inst, err := LoadInstanceConfig(id)
		if err != nil {
			continue
		}
		total += inst.Memory
	}
	return total, nil
}

func TotalEnabledMemory(excludeID string) (int, error) {
	units, err := SDManager.List("mcsd-instance@*.service")
	if err != nil {
		return 0, &ServerError{Message: fmt.Sprintf("list servers: %s", err.Error())}
	}
	var total int
	for _, unit := range units {
		if !SDManager.IsEnabled(UnitToID(unit.Name)) {
			continue
		}
		id := UnitToID(unit.Name)
		if id == excludeID {
			continue
		}
		inst, err := LoadInstanceConfig(id)
		if err != nil {
			continue
		}
		total += inst.Memory
	}
	return total, nil
}

func CheckMemoryBudget(excludeID string, requestedMemory int) error {
	config, err := LoadConfig()
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("load config: %s", err.Error())}
	}
	if config.MemoryBudget <= 0 {
		return nil
	}
	used, err := TotalReservedMemory(excludeID)
	if err != nil {
		return err
	}
	if used+requestedMemory > config.MemoryBudget {
		return &ValidationError{
			Message: fmt.Sprintf("memory budget exceeded: %dMB used + %dMB requested > %dMB budget", used, requestedMemory, config.MemoryBudget),
		}
	}
	return nil
}

func CheckEnableBudget(id string, memory int) error {
	config, err := LoadConfig()
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("load config: %s", err.Error())}
	}
	if config.MemoryBudget <= 0 {
		return nil
	}
	enabled, err := TotalEnabledMemory(id)
	if err != nil {
		return err
	}
	if enabled+memory > config.MemoryBudget {
		return &ValidationError{
			Message: fmt.Sprintf("memory budget exceeded: %dMB enabled + %dMB requested > %dMB budget",
				enabled, memory, config.MemoryBudget),
		}
	}
	return nil
}

func DefaultMemory() int {
	cfg, err := LoadConfig()
	if err != nil || cfg.MemoryBudget <= 0 {
		return DefaultPreferredMemory
	}
	if cfg.MemoryBudget < DefaultPreferredMemory {
		return cfg.MemoryBudget
	}
	return DefaultPreferredMemory
}
