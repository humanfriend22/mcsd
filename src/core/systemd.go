package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbus "github.com/coreos/go-systemd/v22/dbus"
)

var bgCtx = context.Background()

type ServiceStatus struct {
	Name        string
	State       string // active, inactive, failed, activating, deactivating
	SubState    string // running, dead, exited, failed, ...
	Description string
}

type SDClient struct {
	conn *dbus.Conn
}

func NewSDClient() (*SDClient, error) {
	conn, err := dbus.NewSystemdConnectionContext(bgCtx)
	if err != nil {
		return nil, fmt.Errorf("connect to systemd D-Bus: %w", err)
	}
	return &SDClient{conn: conn}, nil
}

func (client *SDClient) Close() {
	client.conn.Close()
}

func (client *SDClient) Start(unit string) error {
	done := make(chan string, 1)
	_, err := client.conn.StartUnitContext(bgCtx, unit, "replace", done)
	if err != nil {
		return fmt.Errorf("start %s: systemd start failed: %w", unit, err)
	}
	if result := <-done; result != "done" {
		return fmt.Errorf("start %s: job result %s", unit, result)
	}
	return nil
}

func (client *SDClient) Stop(unit string) error {
	done := make(chan string, 1)
	_, err := client.conn.StopUnitContext(bgCtx, unit, "replace", done)
	if err != nil {
		return fmt.Errorf("stop %s: systemd stop failed: %w", unit, err)
	}
	if result := <-done; result != "done" && result != "cancelled" {
		return fmt.Errorf("stop %s: job result %s", unit, result)
	}
	return nil
}

func (client *SDClient) Restart(unit string) error {
	done := make(chan string, 1)
	_, err := client.conn.RestartUnitContext(bgCtx, unit, "replace", done)
	if err != nil {
		return fmt.Errorf("restart %s: systemd restart failed: %w", unit, err)
	}
	if result := <-done; result != "done" {
		return fmt.Errorf("restart %s: job result %s", unit, result)
	}
	return nil
}

func (client *SDClient) Status(unit string) (*ServiceStatus, error) {
	statuses, err := client.conn.ListUnitsByNamesContext(bgCtx, []string{unit})
	if err != nil {
		return nil, fmt.Errorf("status %s: systemd status failed: %w", unit, err)
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

func (client *SDClient) List(pattern string) ([]*ServiceStatus, error) {
	units, err := client.conn.ListUnitsByPatternsContext(bgCtx, nil, []string{pattern})
	if err != nil {
		return nil, fmt.Errorf("list %s: systemd list failed: %w", pattern, err)
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

func (client *SDClient) Reload() error {
	if err := client.conn.ReloadContext(bgCtx); err != nil {
		return fmt.Errorf("reload systemd: %w", err)
	}
	return nil
}

func (client *SDClient) Enable(unit string) error {
	_, _, err := client.conn.EnableUnitFilesContext(bgCtx, []string{unit}, false, true)
	if err != nil {
		return fmt.Errorf("enable %s: systemd enable failed: %w", unit, err)
	}
	return nil
}

func (client *SDClient) Disable(unit string) error {
	_, err := client.conn.DisableUnitFilesContext(bgCtx, []string{unit}, false)
	if err != nil {
		return fmt.Errorf("disable %s: systemd disable failed: %w", unit, err)
	}
	return nil
}

// ActiveSince returns when the unit last entered the active state.
// Returns zero time if the unit is not currently active.
func (client *SDClient) ActiveSince(unit string) (time.Time, error) {
	props, err := client.conn.GetUnitPropertiesContext(bgCtx, unit)
	if err != nil {
		return time.Time{}, fmt.Errorf("active since %s: systemd properties failed: %w", unit, err)
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

func (client *SDClient) MemoryUsedBytes(unit string) (uint64, error) {
	props, err := client.conn.GetUnitPropertiesContext(bgCtx, unit)
	if err != nil {
		return 0, fmt.Errorf("memory used %s: systemd properties failed: %w", unit, err)
	}
	v, ok := props["MemoryCurrent"]
	if !ok {
		return 0, nil
	}
	bytes, ok := v.(uint64)
	if !ok || bytes == ^uint64(0) {
		return 0, nil
	}
	return bytes, nil
}

func (client *SDClient) IsEnabled(unit string) bool {
	files, err := client.conn.ListUnitFilesByPatternsContext(bgCtx, []string{"enabled"}, []string{unit})
	if err != nil {
		return false
	}
	return len(files) > 0
}

// UnitName returns the systemd unit name for an instance ID.
func UnitName(id string) string {
	return "mcsd-server@" + id + ".service"
}

// UnitToID extracts the instance ID from a mcsd-server@<id>.service unit name.
func UnitToID(unit string) string {
	s := strings.TrimPrefix(unit, "mcsd-server@")
	return strings.TrimSuffix(s, ".service")
}
