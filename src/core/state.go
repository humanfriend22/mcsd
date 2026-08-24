// Package-internal split: state.go is the live runtime-state read surface —
// the systemd/D-Bus queries and the InstanceState value they assemble.
// Mutating unit commands live in systemd.go.
package core

import (
	"fmt"
	"time"

	. "mcsd/utils"
)

// ServiceStatus is the value produced by a systemd unit status query.
type ServiceStatus struct {
	Name        string
	State       string // active, inactive, failed, activating, deactivating
	SubState    string // running, dead, exited, failed, ...
	Description string
}

// InstanceStateError is the shared wire-level state value a consumer (the
// API DTO, the CLI table writer) assigns to an instance it could not load.
// Core itself never sets it — it is deliberately distinct from systemd's own
// "failed" state so a consumer can tell "config unreadable" apart from
// "process crashed". Kept here, rather than duplicated in src/api and
// src/cli, so those two independent consumers cannot drift on the literal.
const InstanceStateError = "error"

// InstanceState is the live runtime-state snapshot for a single instance,
// gathered from systemd/D-Bus. It is embedded anonymously by Instance so its
// four fields stay flattened to the top level of the instance's JSON payload.
type InstanceState struct {
	State       string     `json:"state"`        // from systemd, InstanceStateError for internal errors
	Enabled     bool       `json:"enabled"`      // systemd service status
	ActiveSince *time.Time `json:"active_since"` // from D-Bus ActiveEnterTimestamp; nil if never active
	MemoryUsed  int        `json:"memory_used"`  // from D-Bus MemoryCurrent (MB)
}

// loadInstanceState gathers the live systemd state for id into an
// InstanceState value. It is unexported deliberately: quick task 260824-05o
// made LoadInstanceConfig the single choke-point for id validation, and an
// exported state loader taking a raw id would hand an unvalidated string to
// UnitName and reopen that gap. LoadInstance is the only caller, and it only
// reaches this after LoadInstanceConfig (and therefore ValidateID) succeeds.
func loadInstanceState(id string) InstanceState {
	state := InstanceState{State: "inactive"}

	if SDManager != nil {
		if status, err := SDManager.Status(id); err == nil {
			state.State = status.State
		}
		state.Enabled = SDManager.IsEnabled(id)

		if since, memMB, err := SDManager.ActiveStats(id); err == nil {
			if !since.IsZero() {
				state.ActiveSince = &since
			}
			state.MemoryUsed = memMB
		}
	}

	return state
}

// Status returns the current systemd service status.
func (instance *Instance) Status() (*ServiceStatus, error) {
	return SDManager.Status(instance.ID)
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
