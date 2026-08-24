// Package-internal split: systemd.go is the D-Bus connection lifecycle and
// unit-command surface — opening/closing the connection, naming units, and
// issuing the mutating commands (Start/Stop/Restart/Enable/Disable/
// ResetFailed/Reload). Live-state reads live in state.go.
package core

import (
	"context"
	"fmt"
	"strings"

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
