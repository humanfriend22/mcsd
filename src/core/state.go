// Package-internal split: state.go is the InstanceState value type and its
// assembly from live systemd state. All systemd/D-Bus methods live in
// systemd.go.
package core

import "time"

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
