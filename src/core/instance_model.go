package core

import (
	"fmt"
	"os"
	"time"

	"mcsd/vendors"

	. "mcsd/utils"
)

// Instance is the master struct representing everything about a Minecraft server instance.
// It composes the persisted config, the server.properties ports, and the live systemd state.
type Instance struct {
	*InstanceConfig                    // config.json fields (flattened via embedding)
	Ports         Ports  `json:"ports"`           // from server.properties
	State         string `json:"state"`           // from systemd: "active", "inactive", "failed", etc.
	Enabled       bool   `json:"enabled"`         // from systemd
	UptimeSeconds int    `json:"uptime_seconds"`  // computed from ActiveEnterTimestamp
	MemoryUsed    int    `json:"memory_used"`     // from D-Bus MemoryCurrent (MB)
}

// LoadInstance builds a fully-populated Instance from config.json, server.properties, and systemd.
func LoadInstance(id string) (*Instance, error) {
	cfg, err := LoadInstanceConfig(id)
	if err != nil {
		return nil, err
	}

	ports, err := ReadPorts(InstanceDir(id))
	if err != nil {
		ports = Ports{} // server.properties may not exist yet
	}

	state := "inactive"
	var enabled bool
	var uptimeSeconds int
	var memoryUsed int

	if SDManager != nil {
		status, err := SDManager.Status(id)
		if err == nil {
			state = status.State
		}
		enabled = SDManager.IsEnabled(id)

		if since, memMB, err := SDManager.ActiveStats(id); err == nil {
			if !since.IsZero() {
				uptimeSeconds = int(time.Since(since).Seconds())
			}
			memoryUsed = memMB
		}
	}

	return &Instance{
		InstanceConfig: cfg,
		Ports:          ports,
		State:          state,
		Enabled:        enabled,
		UptimeSeconds:  uptimeSeconds,
		MemoryUsed:     memoryUsed,
	}, nil
}

// ListInstances returns all instances with fully-populated data.
func ListInstances() ([]*Instance, error) {
	ids, err := ListInstanceConfigs()
	if err != nil {
		return nil, err
	}
	var instances []*Instance
	for _, id := range ids {
		inst, err := LoadInstance(id)
		if err != nil {
			continue
		}
		instances = append(instances, inst)
	}
	return instances, nil
}

// Validate checks the instance config for correctness.
func (inst *Instance) Validate() error {
	return inst.InstanceConfig.Validate()
}

// NewInstance validates the config and ports, checks ID uniqueness and memory budget,
// and returns an Instance ready for Create().
func NewInstance(config InstanceConfig, ports Ports) (*Instance, error) {
	inst := &Instance{InstanceConfig: &config}
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	if err := ValidateIDUnique(inst.ID); err != nil {
		return nil, err
	}
	if err := ports.Validate(); err != nil {
		return nil, err
	}
	if err := CheckMemoryBudget(inst.ID, inst.Memory); err != nil {
		return nil, err
	}
	return inst, nil
}

// Create provisions a new instance: writes config.json, server.properties, eula.txt.
func (inst *Instance) Create(ports Ports) error {
	if err := os.MkdirAll(InstanceDir(inst.ID), 0750); err != nil {
		return &ServerError{Message: fmt.Sprintf("create instance dir: %s", err.Error())}
	}

	var provisionErr error
	defer func() {
		if provisionErr != nil {
			_ = os.RemoveAll(InstanceDir(inst.ID))
		}
	}()

	if provisionErr = WriteInstanceConfig(inst.ID, inst.InstanceConfig); provisionErr != nil {
		return &ServerError{Message: fmt.Sprintf("write instance config: %s", provisionErr.Error())}
	}

	if provisionErr = WriteServerProperties(InstanceDir(inst.ID), ports); provisionErr != nil {
		return &ServerError{Message: fmt.Sprintf("write server.properties: %s", provisionErr.Error())}
	}

	if provisionErr = WriteAtomic(InstanceDir(inst.ID)+"/eula.txt", []byte("eula=true\n"), 0640); provisionErr != nil {
		return &ServerError{Message: fmt.Sprintf("write eula.txt: %s", provisionErr.Error())}
	}

	return nil
}

// DeleteInstance disables, resets, and removes an instance by ID.
// Used by orphan cleanup which may not have a full Instance.
func DeleteInstance(id string) error {
	if SDManager == nil {
		return &ServerError{Message: "systemd not initialized"}
	}

	// Reset failed state while unit is still loaded.
	_ = SDManager.ResetFailed(id)

	// Safety net: stop if still active. Shouldn't happen via UI.
	if status, err := SDManager.Status(id); err == nil && status.State == "active" {
		_ = SDManager.Stop(id)
	}

	_ = SDManager.Disable(id)

	if _, err := os.Stat(InstanceDir(id)); !os.IsNotExist(err) {
		if err := os.RemoveAll(InstanceDir(id)); err != nil {
			return &ServerError{Message: fmt.Sprintf("remove instance dir: %s", err.Error())}
		}
	}
	return SDManager.Reload()
}

// Download fetches the server JAR to the instance directory.
func (inst *Instance) Download(url string) error {
	executable := vendors.Executable(inst.Vendor)
	return vendors.DownloadFile(InstanceDir(inst.ID)+"/"+executable, url)
}

func (inst *Instance) Upgrade(v vendors.Vendor, version string, build int) error {
	status, err := SDManager.Status(inst.ID)
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("check status: %s", err.Error())}
	}
	if status.State == "active" {
		return &ValidationError{Message: "server must be stopped before upgrading"}
	}

	downloadURL, err := v.DownloadURL(version, build)
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("resolve download URL: %s", err.Error())}
	}
	if err := inst.Download(downloadURL); err != nil {
		return &ServerError{Message: fmt.Sprintf("download: %s", err.Error())}
	}

	inst.Vendor = v.Name()
	inst.Version = version
	inst.Build = build
	if err := WriteInstanceConfig(inst.ID, inst.InstanceConfig); err != nil {
		return &ServerError{Message: fmt.Sprintf("write instance config: %s", err.Error())}
	}
	return nil
}

// EnsureStartReady checks that the instance can start (port availability + memory budget).
func (inst *Instance) EnsureStartReady() error {
	ports, err := ReadPorts(InstanceDir(inst.ID))
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("read ports: %s", err.Error())}
	}
	if err := checkPortAvailable(ports.Game); err != nil {
		return &ValidationError{Message: fmt.Sprintf("game port: %s", err.Error())}
	}
	if err := checkPortAvailable(ports.RCON); err != nil {
		return &ValidationError{Message: fmt.Sprintf("RCON port: %s", err.Error())}
	}
	if err := CheckMemoryBudget(inst.ID, inst.Memory); err != nil {
		return err
	}
	return nil
}

// Start validates readiness then starts the instance via systemd.
func (inst *Instance) Start() error {
	if err := inst.EnsureStartReady(); err != nil {
		return err
	}
	return SDManager.Start(inst.ID)
}

// Stop stops the instance via systemd.
func (inst *Instance) Stop() error {
	return SDManager.Stop(inst.ID)
}

// Restart restarts the instance via systemd.
func (inst *Instance) Restart() error {
	return SDManager.Restart(inst.ID)
}

// Status returns the current systemd service status.
func (inst *Instance) Status() (*ServiceStatus, error) {
	return SDManager.Status(inst.ID)
}

// Enable enables the instance to start on boot, checking memory budget first.
func (inst *Instance) Enable() error {
	if err := CheckEnableBudget(inst.ID, inst.Memory); err != nil {
		return err
	}
	if err := SDManager.Enable(inst.ID); err != nil {
		return &ValidationError{Message: fmt.Sprintf("enable failed: %s", err.Error())}
	}
	return SDManager.Reload()
}

// Disable disables the instance from starting on boot.
func (inst *Instance) Disable() error {
	if err := SDManager.Disable(inst.ID); err != nil {
		return err
	}
	return SDManager.Reload()
}

// RCON sends a command to the running server via the RCON protocol.
func (inst *Instance) RCON(cmd string) (string, error) {
	ports, err := ReadPorts(InstanceDir(inst.ID))
	if err != nil {
		return "", err
	}
	rconClient, err := DialRCON(fmt.Sprintf("127.0.0.1:%d", ports.RCON), ports.RCONPassword)
	if err != nil {
		return "", err
	}
	defer rconClient.Close()
	return rconClient.Send(cmd)
}

type PatchRequest struct {
	Name       *string  `json:"name"`
	Vendor     *string  `json:"vendor"`
	Version    *string  `json:"version"`
	Binary     *string  `json:"binary"`
	JavaArgs   []string `json:"java_args"`
	ServerArgs []string `json:"server_args"`
	Memory     *int     `json:"memory"`
	GamePort   *int     `json:"game_port"`
	RCONPort   *int     `json:"rcon_port"`
}

func (inst *Instance) Patch(req PatchRequest) error {
	if req.Name != nil {
		inst.Name = *req.Name
	}
	if req.Vendor != nil {
		inst.Vendor = *req.Vendor
	}
	if req.Version != nil {
		inst.Version = *req.Version
	}
	if req.Binary != nil {
		inst.Binary = *req.Binary
	}
	if req.JavaArgs != nil {
		inst.JavaArgs = req.JavaArgs
	}
	if req.ServerArgs != nil {
		inst.ServerArgs = req.ServerArgs
	}
	if req.Memory != nil {
		inst.Memory = *req.Memory
	}

	if err := inst.Validate(); err != nil {
		return err
	}
	if err := WriteInstanceConfig(inst.ID, inst.InstanceConfig); err != nil {
		return &ServerError{Message: fmt.Sprintf("write instance config: %s", err.Error())}
	}

	if req.GamePort != nil || req.RCONPort != nil {
		status, err := SDManager.Status(inst.ID)
		if err != nil {
			return &ServerError{Message: fmt.Sprintf("check status: %s", err.Error())}
		}
		if status.State == "active" {
			return &ValidationError{Message: "stop the server before changing ports"}
		}

		ports := inst.Ports
		if req.GamePort != nil {
			ports.Game = *req.GamePort
		}
		if req.RCONPort != nil {
			ports.RCON = *req.RCONPort
		}
		if err := ports.Validate(); err != nil {
			return err
		}
		if err := checkPortAvailable(ports.Game); err != nil {
			return err
		}
		if err := checkPortAvailable(ports.RCON); err != nil {
			return err
		}
		if err := WriteServerProperties(InstanceDir(inst.ID), ports); err != nil {
			return &ServerError{Message: fmt.Sprintf("write server.properties: %s", err.Error())}
		}
	}

	return nil
}
