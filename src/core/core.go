package core

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	. "mcsd/utils"
)

//go:embed services/mcsd.service
var DaemonServiceContent string

//go:embed services/mcsd-instance@.service
var DaemonServiceTemplateContent string

//go:embed services/override.conf.tmpl
var InstanceOverrideTemplateContent string

const (
	GlobalConfigPath = "/srv/mcsd/config.json"

	DaemonServicePath         = "/etc/systemd/system/mcsd.service"
	DaemonServiceTemplatePath = "/etc/systemd/system/mcsd-instance@.service"

	DaemonDropInDirectory = "/etc/systemd/system/%s.d"
)

// instanceOverrideDir returns the systemd drop-in directory for the given
// instance's templated unit (mcsd-instance@<id>.service.d).
func instanceOverrideDir(id string) string {
	return fmt.Sprintf(DaemonDropInDirectory, UnitName(id))
}

// instanceOverridePath returns the drop-in file inside instanceOverrideDir.
func instanceOverridePath(id string) string {
	return filepath.Join(instanceOverrideDir(id), "override.conf")
}

// InstanceOverrideExists reports whether the per-instance drop-in (which
// carries MemoryMax) has been written to disk.
func InstanceOverrideExists(id string) bool {
	_, err := os.Stat(instanceOverridePath(id))
	return err == nil
}

// WriteInstanceOverride renders the override template with the instance's
// memory limit and writes it as a systemd drop-in, then reloads systemd so
// the unit picks it up.
func WriteInstanceOverride(id string, memoryMB int) error {
	tmpl, err := template.New("override").Parse(InstanceOverrideTemplateContent)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("parse override template: %s", err.Error())}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ MemoryMB int }{memoryMB}); err != nil {
		return &InternalError{Message: fmt.Sprintf("render override template: %s", err.Error())}
	}

	if err := os.MkdirAll(instanceOverrideDir(id), 0755); err != nil {
		return &InternalError{Message: fmt.Sprintf("create override dir: %s", err.Error())}
	}
	if err := WriteFile(instanceOverridePath(id), buf.Bytes(), 0644); err != nil {
		return &InternalError{Message: fmt.Sprintf("write override file: %s", err.Error())}
	}

	return SDManager.Reload()
}

// DeleteInstanceOverride removes the instance's drop-in directory and
// reloads systemd. Removing a non-existent directory is a no-op error-wise.
func DeleteInstanceOverride(id string) error {
	if err := os.RemoveAll(instanceOverrideDir(id)); err != nil {
		return &InternalError{Message: fmt.Sprintf("remove override dir: %s", err.Error())}
	}
	return SDManager.Reload()
}

// Sets up mcsd on this host. memoryBudget is the RAM (MB) to reserve across
// all instances; 0 falls back to all system RAM minus 512 MB. Returns the resolved budget.
func Init(memoryBudget int) (int, error) {
	total, err := TotalSystemMemory()
	if err != nil {
		return 0, &InternalError{Message: fmt.Sprintf("read system memory: %s", err.Error())}
	}

	if memoryBudget == 0 {
		memoryBudget = total - 512
	}

	if memoryBudget < 512 {
		return 0, &ValidationError{Message: fmt.Sprintf("memory budget must be at least 512 MB, got %d", memoryBudget)}
	}
	if total-memoryBudget < 512 {
		return 0, &ValidationError{Message: fmt.Sprintf("memory budget %d MB leaves less than 512 MB for system (total: %d MB)", memoryBudget, total)}
	}

	if err := os.MkdirAll(DefaultBasePath, 0755); err != nil {
		return 0, &InternalError{Message: fmt.Sprintf("create instances dir: %s", err.Error())}
	}

	config := &Config{MemoryBudget: memoryBudget}
	if err := WriteConfig(config); err != nil {
		return 0, &InternalError{Message: fmt.Sprintf("write config: %s", err.Error())}
	}

	if err := os.WriteFile(DaemonServicePath, []byte(DaemonServiceContent), 0644); err != nil {
		return 0, &InternalError{Message: fmt.Sprintf("write daemon service: %s", err.Error())}
	}

	if err := os.WriteFile(DaemonServiceTemplatePath, []byte(DaemonServiceTemplateContent), 0644); err != nil {
		return 0, &InternalError{Message: fmt.Sprintf("write daemon service template: %s", err.Error())}
	}

	if err := SDManager.Enable(""); err != nil {
		return 0, err
	}

	if err := SDManager.Reload(); err != nil {
		return 0, err
	}

	return memoryBudget, nil
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
		return &InternalError{Message: fmt.Sprintf("reload systemd: %s", err.Error())}
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
		return &InternalError{Message: fmt.Sprintf("mcsd not fully initialized:\n- %s", strings.Join(missing, "\n- "))}
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
