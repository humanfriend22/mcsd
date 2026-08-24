// systemd.go: all systemd/D-Bus methods for sdManager — connection
// lifecycle, unit naming, mutating commands (Start/Stop/Restart/Enable/
// Disable/ResetFailed/Reload), and state queries (Status/IsEnabled/
// ActiveStats/ActiveSince/List). state.go holds the InstanceState value
// type these queries assemble into and the Instance-level accessor.
package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"

	. "mcsd/utils"
)

var bgCtx = context.Background()

var SDManager *sdManager // initialized once per process by InitSDManager

type sdManager struct {
	conn   *dbus.Conn
	rawBus *godbus.Conn
}

// ServiceStatus is the value produced by a systemd unit status query.
type ServiceStatus struct {
	Name        string
	State       string // active, inactive, failed, activating, deactivating
	SubState    string // running, dead, exited, failed, ...
	Description string
}

// InitSDManager opens the D-Bus connection to systemd.
// Must be called once at process entry before any other core function.
func InitSDManager() error {
	conn, err := dbus.NewSystemdConnectionContext(bgCtx)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("connect to systemd D-Bus: %s", err.Error())}
	}
	rawBus, err := godbus.SystemBus()
	if err != nil {
		conn.Close()
		return &InternalError{Message: fmt.Sprintf("connect to system D-Bus: %s", err.Error())}
	}
	SDManager = &sdManager{conn: conn, rawBus: rawBus}
	return nil
}

// ShutdownSDManager closes the D-Bus connection.
func ShutdownSDManager() {
	if SDManager != nil {
		SDManager.conn.Close()
		SDManager.rawBus.Close()
		SDManager = nil
	}
}

// UnitName returns the systemd unit name for an instance ID.
// Empty ID maps to "mcsd.service".
func UnitName(id string) string {
	if id == "" {
		return "mcsd.service"
	}
	return "mcsd-instance@" + id + ".service"
}

// UnitToID extracts the instance ID from a mcsd-instance@<id>.service unit name.
func UnitToID(unit string) string {
	s := strings.TrimPrefix(unit, "mcsd-instance@")
	return strings.TrimSuffix(s, ".service")
}

// Start starts the unit for the given instance ID.
// If id is empty, starts mcsd.service.
func (s *sdManager) Start(id string) error {
	done := make(chan string, 1)
	_, err := s.conn.StartUnitContext(bgCtx, UnitName(id), "replace", done)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("start %s: systemd start failed: %s", id, err.Error())}
	}
	if result := <-done; result != "done" {
		return &InternalError{Message: fmt.Sprintf("start %s: job result %s", id, result)}
	}
	return nil
}

// Stop stops the unit for the given instance ID.
// If id is empty, stops mcsd.service.
func (s *sdManager) Stop(id string) error {
	done := make(chan string, 1)
	_, err := s.conn.StopUnitContext(bgCtx, UnitName(id), "replace", done)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("stop %s: systemd stop failed: %s", id, err.Error())}
	}
	if result := <-done; result != "done" && result != "cancelled" {
		return &InternalError{Message: fmt.Sprintf("stop %s: job result %s", id, result)}
	}
	return nil
}

// Restart restarts the unit for the given instance ID.
// If id is empty, restarts mcsd.service.
func (s *sdManager) Restart(id string) error {
	done := make(chan string, 1)
	_, err := s.conn.RestartUnitContext(bgCtx, UnitName(id), "replace", done)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("restart %s: systemd restart failed: %s", id, err.Error())}
	}
	if result := <-done; result != "done" {
		return &InternalError{Message: fmt.Sprintf("restart %s: job result %s", id, result)}
	}
	return nil
}

// Enable enables the unit for the given instance ID for boot.
// If id is empty, enables mcsd.service.
func (s *sdManager) Enable(id string) error {
	_, _, err := s.conn.EnableUnitFilesContext(bgCtx, []string{UnitName(id)}, false, true)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("enable %s: systemd enable failed: %s", id, err.Error())}
	}
	return nil
}

// Disable disables the unit for the given instance ID from boot.
// If id is empty, disables mcsd.service.
func (s *sdManager) Disable(id string) error {
	_, err := s.conn.DisableUnitFilesContext(bgCtx, []string{UnitName(id)}, false)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("disable %s: systemd disable failed: %s", id, err.Error())}
	}
	return nil
}

// ResetFailed clears the failed state of the unit for the given instance ID.
// If id is empty, resets mcsd.service.
func (s *sdManager) ResetFailed(id string) error {
	return s.conn.ResetFailedUnitContext(bgCtx, UnitName(id))
}

// Reload reloads the systemd manager configuration.
func (s *sdManager) Reload() error {
	if err := s.conn.ReloadContext(bgCtx); err != nil {
		return &InternalError{Message: fmt.Sprintf("reload systemd: %s", err.Error())}
	}
	return nil
}

// Status returns the current service status for the given instance ID.
// If id is empty, returns status of mcsd.service.
func (s *sdManager) Status(id string) (*ServiceStatus, error) {
	unit := UnitName(id)
	statuses, err := s.conn.ListUnitsByNamesContext(bgCtx, []string{unit})
	if err != nil {
		return nil, &InternalError{Message: fmt.Sprintf("status %s: systemd status failed: %s", id, err.Error())}
	}
	if len(statuses) == 0 {
		return &ServiceStatus{Name: unit, State: "inactive", SubState: "dead"}, nil
	}
	status := statuses[0]
	return &ServiceStatus{
		Name:        unit,
		State:       status.ActiveState,
		SubState:    status.SubState,
		Description: status.Description,
	}, nil
}

// IsEnabled returns whether the unit for the given instance ID is enabled at boot.
// If id is empty, checks mcsd.service.
// Calls GetUnitFileState via raw D-Bus, which works for template instances unlike
// ListUnitFilesByPatterns.
func (s *sdManager) IsEnabled(id string) bool {
	unit := UnitName(id)
	var state string
	obj := s.rawBus.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	err := obj.Call("org.freedesktop.systemd1.Manager.GetUnitFileState", 0, unit).Store(&state)
	if err != nil {
		return false
	}
	return state == "enabled"
}

// ActiveStats returns the active-enter timestamp and memory usage (MB) for the given unit.
func (s *sdManager) ActiveStats(id string) (activeSince time.Time, memoryMB int, err error) {
	unitName := UnitName(id)

	// Unit-level properties
	props, err := s.conn.GetUnitPropertiesContext(bgCtx, unitName)
	if err != nil {
		return time.Time{}, 0, &InternalError{Message: fmt.Sprintf("active stats %s: unit properties failed: %s", id, err.Error())}
	}
	if ts, ok := props["ActiveEnterTimestamp"].(uint64); ok && ts > 0 {
		activeSince = time.UnixMicro(int64(ts))
	}

	// Service-level properties (MemoryCurrent lives on org.freedesktop.systemd1.Service)
	svcProps, err := s.conn.GetUnitTypePropertiesContext(bgCtx, unitName, "Service")
	if err == nil {
		if v, ok := svcProps["MemoryCurrent"].(uint64); ok && v != ^uint64(0) {
			memoryMB = int(v / (1024 * 1024))
		}
	}

	return activeSince, memoryMB, nil
}

// ActiveSince returns when the unit last entered the active state.
// Returns zero time if the unit is not currently active.
func (s *sdManager) ActiveSince(id string) (time.Time, error) {
	props, err := s.conn.GetUnitPropertiesContext(bgCtx, UnitName(id))
	if err != nil {
		return time.Time{}, &InternalError{Message: fmt.Sprintf("active since %s: systemd properties failed: %s", id, err.Error())}
	}
	timestamp, ok := props["ActiveEnterTimestamp"]
	if !ok {
		return time.Time{}, nil
	}
	microseconds, ok := timestamp.(uint64)
	if !ok || microseconds == 0 {
		return time.Time{}, nil
	}
	return time.UnixMicro(int64(microseconds)), nil
}

// List returns all units matching the given glob pattern.
func (s *sdManager) List(pattern string) ([]*ServiceStatus, error) {
	units, err := s.conn.ListUnitsByPatternsContext(bgCtx, nil, []string{pattern})
	if err != nil {
		return nil, &InternalError{Message: fmt.Sprintf("list %s: systemd list failed: %s", pattern, err.Error())}
	}
	statuses := make([]*ServiceStatus, 0, len(units))
	for _, unit := range units {
		statuses = append(statuses, &ServiceStatus{
			Name:        unit.Name,
			State:       unit.ActiveState,
			SubState:    unit.SubState,
			Description: unit.Description,
		})
	}
	return statuses, nil
}
