package core

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed services/mcsd.service
var DaemonServiceContent string

//go:embed services/mcsd-server@.service
var DaemonServiceTemplateContent string

const (
	DefaultBasePath  = "/srv/mcsd/instances"
	GlobalConfigPath = "/srv/mcsd/config.json"

	DaemonServicePath         = "/etc/systemd/system/mcsd.service"
	DaemonServiceTemplatePath = "/etc/systemd/system/mcsd-server@.service"
)

func InstanceDir(id string) string {
	return filepath.Join(DefaultBasePath, id)
}

func Init(memoryBudget int) error {
	if memoryBudget < 512 {
		return fmt.Errorf("memory budget must be at least 512 MB, got %d", memoryBudget)
	}

	total, err := TotalSystemMemory()
	if err != nil {
		return fmt.Errorf("read system memory: %w", err)
	}
	if total-memoryBudget < 1024 {
		return fmt.Errorf("memory budget %d MB leaves less than 1 GB for system (total: %d MB)", memoryBudget, total)
	}

	if err := os.MkdirAll(DefaultBasePath, 0755); err != nil {
		return fmt.Errorf("create instances dir: %w", err)
	}

	config := &Config{MemoryBudget: memoryBudget}
	if err := WriteConfig(config); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := os.WriteFile(DaemonServicePath, []byte(DaemonServiceContent), 0644); err != nil {
		return fmt.Errorf("write daemon service: %w", err)
	}

	if err := os.WriteFile(DaemonServiceTemplatePath, []byte(DaemonServiceTemplateContent), 0644); err != nil {
		return fmt.Errorf("write daemon service template: %w", err)
	}

	return nil
}

func DeInit(client *SDClient) error {
	if ids, _ := ListInstanceConfigs(); len(ids) > 0 {
		return fmt.Errorf(
			"%d instance(s) still exist: %s\nDelete them first with 'mcsd delete <id>'",
			len(ids), strings.Join(ids, ", "),
		)
	}

	if units, err := client.List("mcsd-server@*.service"); err == nil {
		for _, unit := range units {
			if unit.State == "active" {
				_ = client.Stop(unit.Name)
			}
			_ = client.Disable(unit.Name)
		}
	}

	if status, err := client.Status("mcsd.service"); err == nil && status.State == "active" {
		_ = client.Stop("mcsd.service")
	}
	_ = client.Disable("mcsd.service")

	_ = os.Remove(DaemonServicePath)
	_ = os.Remove(DaemonServiceTemplatePath)
	_ = os.RemoveAll(filepath.Dir(GlobalConfigPath))

	return client.Reload()
}

// EnsureReady verifies mcsd is fully installed and removes orphaned systemd units.
func EnsureReady() error {
	var missing []string

	if _, err := os.Stat(DefaultBasePath); os.IsNotExist(err) {
		missing = append(missing, "  instances dir missing")
	}
	if _, err := os.Stat(GlobalConfigPath); os.IsNotExist(err) {
		missing = append(missing, "  config missing")
	}
	if _, err := os.Stat(DaemonServicePath); os.IsNotExist(err) {
		missing = append(missing, "  daemon service missing")
	}
	if _, err := os.Stat(DaemonServiceTemplatePath); os.IsNotExist(err) {
		missing = append(missing, "  server template missing")
	}

	if len(missing) > 0 {
		return fmt.Errorf("mcsd not fully initialized — run 'mcsd init':\n%s", strings.Join(missing, "\n"))
	}

	sd, err := NewSDClient()
	if err != nil {
		return fmt.Errorf("systemd unavailable: %w", err)
	}
	defer sd.Close()
	cleanOrphans(sd)
	return nil
}

func cleanOrphans(sd *SDClient) {
	units, err := sd.List("mcsd-server@*.service")
	if err != nil {
		return
	}
	for _, unit := range units {
		id := UnitToID(unit.Name)
		if _, err := os.Stat(InstanceDir(id)); os.IsNotExist(err) {
			_ = DeleteByID(id, sd)
		}
	}
}
