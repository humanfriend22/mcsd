package core

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "mcsd/utils"
)

//go:embed services/mcsd.service
var DaemonServiceContent string

//go:embed services/mcsd-instance@.service
var DaemonServiceTemplateContent string

const (
	DefaultBasePath  = "/srv/mcsd/instances"
	GlobalConfigPath = "/srv/mcsd/config.json"

	DaemonServicePath         = "/etc/systemd/system/mcsd.service"
	DaemonServiceTemplatePath = "/etc/systemd/system/mcsd-instance@.service"
)

func InstanceDir(id string) string {
	return filepath.Join(DefaultBasePath, id)
}

func Init(memoryBudget int) error {
	if memoryBudget < 512 {
		return &ValidationError{Message: fmt.Sprintf("memory budget must be at least 512 MB, got %d", memoryBudget)}
	}

	total, err := TotalSystemMemory()
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("read system memory: %s", err.Error())}
	}
	if total-memoryBudget < 512 {
		return &ValidationError{Message: fmt.Sprintf("memory budget %d MB leaves less than 512 MB for system (total: %d MB)", memoryBudget, total)}
	}

	if err := os.MkdirAll(DefaultBasePath, 0755); err != nil {
		return &ServerError{Message: fmt.Sprintf("create instances dir: %s", err.Error())}
	}

	config := &Config{MemoryBudget: memoryBudget}
	if err := WriteConfig(config); err != nil {
		return &ServerError{Message: fmt.Sprintf("write config: %s", err.Error())}
	}

	if err := os.WriteFile(DaemonServicePath, []byte(DaemonServiceContent), 0644); err != nil {
		return &ServerError{Message: fmt.Sprintf("write daemon service: %s", err.Error())}
	}

	if err := os.WriteFile(DaemonServiceTemplatePath, []byte(DaemonServiceTemplateContent), 0644); err != nil {
		return &ServerError{Message: fmt.Sprintf("write daemon service template: %s", err.Error())}
	}

	return nil
}

func DeInit() error {
	if ids, _ := ListInstanceConfigs(); len(ids) > 0 {
		return &ValidationError{
			Message: fmt.Sprintf("%d instance(s) still exist: %s\nDelete them first with 'mcsd delete <id>'",
				len(ids), strings.Join(ids, ", ")),
		}
	}

	if units, err := SDManager.List("mcsd-instance@*.service"); err == nil {
		for _, unit := range units {
			if unit.State == "active" {
				_ = SDManager.Stop(UnitToID(unit.Name))
			}
			_ = SDManager.Disable(UnitToID(unit.Name))
		}
	}

	if status, err := SDManager.Status(""); err == nil && status.State == "active" {
		_ = SDManager.Stop("")
	}
	_ = SDManager.Disable("")

	_ = os.Remove(DaemonServicePath)
	_ = os.Remove(DaemonServiceTemplatePath)
	_ = os.RemoveAll(filepath.Dir(GlobalConfigPath))

	if err := SDManager.Reload(); err != nil {
		return &ServerError{Message: fmt.Sprintf("reload systemd: %s", err.Error())}
	}
	return nil
}

// EnsureReady verifies mcsd is fully installed and removes orphaned systemd units.
func EnsureReady() error {
	var missing []string

	if _, err := os.Stat(DefaultBasePath); os.IsNotExist(err) {
		missing = append(missing, "instances dir missing")
	}
	if _, err := os.Stat(GlobalConfigPath); os.IsNotExist(err) {
		missing = append(missing, "config missing")
	}
	if _, err := os.Stat(DaemonServicePath); os.IsNotExist(err) {
		missing = append(missing, "daemon service missing")
	}
	if _, err := os.Stat(DaemonServiceTemplatePath); os.IsNotExist(err) {
		missing = append(missing, "server template missing")
	}

	if len(missing) > 0 {
		return &ServerError{Message: fmt.Sprintf("mcsd not fully initialized:\n- %s", strings.Join(missing, "\n- "))}
	}

	cleanOrphans()
	return nil
}

func cleanOrphans() {
	units, err := SDManager.List("mcsd-instance@*.service")
	if err != nil {
		return
	}
	for _, unit := range units {
		id := UnitToID(unit.Name)
		if _, err := os.Stat(InstanceDir(id)); os.IsNotExist(err) {
			_ = DeleteInstance(id)
		}
	}
}
