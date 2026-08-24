package core

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"mcsd/vendors"

	. "mcsd/utils"
)

const InstanceStateError = "error"

// Master struct for a single server
type Instance struct {
	*InstanceConfig            // config.json fields (flattened via embedding)
	Ports           Ports      `json:"ports"`        // from server.properties
	State           string     `json:"state"`        // from systemd, InstanceStateError for internal errors
	Enabled         bool       `json:"enabled"`
	ActiveSince     *time.Time `json:"active_since"` // from D-Bus ActiveEnterTimestamp; nil if never active
	MemoryUsed      int        `json:"memory_used"`  // from D-Bus MemoryCurrent (MB)
}

// LoadInstance builds a fully-populated Instance from config.json, server.properties, and systemd.
func LoadInstance(id string) (*Instance, error) {
	config, err := LoadInstanceConfig(id)
	if err != nil {
		return nil, err
	}

	state := "inactive"
	var enabled bool
	var activeSince *time.Time
	var memoryUsed int

	if SDManager != nil {
		status, err := SDManager.Status(id)
		if err == nil {
			state = status.State
		}
		enabled = SDManager.IsEnabled(id)

		if since, memMB, err := SDManager.ActiveStats(id); err == nil {
			if !since.IsZero() {
				activeSince = &since
			}
			memoryUsed = memMB
		}
	}

	instance := Instance{
		InstanceConfig: config,
		State:          state,
		Enabled:        enabled,
		ActiveSince:    activeSince,
		MemoryUsed:     memoryUsed,
	}

	ports, err := instance.ReadPorts()
	if err != nil {
		return nil, err
	}
	instance.Ports = ports

	return &instance, nil
}

// InstanceResult pairs an instance id with either its successfully loaded
// Instance or the error that occurred while loading it. Exactly one of
// Instance / Err is non-nil. ID is always populated so a consumer that only
// needs identity (e.g. src/api/cache.go's closeStaleRCON) never has to
// dereference Instance — core fabricates no stand-in domain object for a
// load failure; each consumer builds its own error representation.
type InstanceResult struct {
	ID       string
	Instance *Instance
	Err      error
}

var (
	loadFailureMu   sync.Mutex
	lastLoadFailure = map[string]string{}
)

// logInstanceLoadFailure records a per-instance load failure, deduped so an
// unchanged repeated failure is logged once rather than once per
// ListInstances call (src/api/cache.go polls every 2 seconds).
func logInstanceLoadFailure(id string, err error) {
	msg := err.Error()

	loadFailureMu.Lock()
	defer loadFailureMu.Unlock()

	if lastLoadFailure[id] == msg {
		return
	}
	lastLoadFailure[id] = msg
	log.Printf("instance %q failed to load: %s", id, msg)
}

// clearInstanceLoadFailure drops the dedupe entry for id so a recurrence
// after a repair is logged again.
func clearInstanceLoadFailure(id string) {
	loadFailureMu.Lock()
	defer loadFailureMu.Unlock()
	delete(lastLoadFailure, id)
}

// buildInstanceList walks ids in order and appends exactly one InstanceResult
// per id: a success entry carrying the loaded instance, or a failure entry
// carrying the load error unwrapped — never a fabricated stand-in Instance.
// Taking the loader as a parameter is the seam that makes this testable
// without the compile-time DefaultBasePath.
func buildInstanceList(ids []string, load func(string) (*Instance, error)) []InstanceResult {
	results := make([]InstanceResult, 0, len(ids))
	for _, id := range ids {
		inst, err := load(id)
		if err != nil {
			logInstanceLoadFailure(id, err)
			results = append(results, InstanceResult{ID: id, Err: err})
			continue
		}
		clearInstanceLoadFailure(id)
		results = append(results, InstanceResult{ID: id, Instance: inst})
	}
	return results
}

// ListInstances returns one InstanceResult per known instance id. A
// per-instance load failure surfaces as that entry's Err rather than
// shortening the list or failing the whole call (D-01); the returned error
// is non-nil only when listing the ids themselves fails (e.g. the instances
// directory is unreadable), which is a whole-list failure distinct from an
// empty list.
func ListInstances() ([]InstanceResult, error) {
	ids, err := ListInstanceConfigs()
	if err != nil {
		return nil, err
	}
	return buildInstanceList(ids, LoadInstance), nil
}

// Validate checks the instance config, ports, and memory budget for correctness.
func (instance *Instance) Validate() error {
	if err := instance.InstanceConfig.Validate(); err != nil {
		return err
	}
	if err := instance.Ports.Validate(); err != nil {
		return err
	}
	return CheckMemoryBudget(instance.ID, instance.Memory)
}

// Create provisions a new instance: writes config.json, server.properties, eula.txt.
func (instance *Instance) Create(ports Ports) error {
	if err := ValidateIDUnique(instance.ID); err != nil {
		return err
	}
	if err := os.MkdirAll(InstanceDir(instance.ID), 0750); err != nil {
		return &InternalError{Message: fmt.Sprintf("create instance dir: %s", err.Error())}
	}

	var provisionErr error
	defer func() {
		if provisionErr != nil {
			_ = os.RemoveAll(InstanceDir(instance.ID))
		}
	}()

	if provisionErr = WriteInstanceConfig(instance.ID, instance.InstanceConfig); provisionErr != nil {
		return &InternalError{Message: fmt.Sprintf("write instance config: %s", provisionErr.Error())}
	}

	if provisionErr = instance.WritePorts(ports); provisionErr != nil {
		return &InternalError{Message: fmt.Sprintf("write ports to server.properties: %s", provisionErr.Error())}
	}

	if provisionErr = WriteFile(InstanceDir(instance.ID)+"/eula.txt", []byte("eula=true\n"), 0640); provisionErr != nil {
		return &InternalError{Message: fmt.Sprintf("write eula.txt: %s", provisionErr.Error())}
	}

	return nil
}

// DeleteInstance disables, resets, and removes an instance by ID.
// Used by orphan cleanup which may not have a full Instance.
func DeleteInstance(id string) error {
	if SDManager == nil {
		return &InternalError{Message: "systemd not initialized"}
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
			return &InternalError{Message: fmt.Sprintf("remove instance dir: %s", err.Error())}
		}
	}
	return SDManager.Reload()
}

// Download fetches the server JAR to the instance directory.
func (instance *Instance) Download(url string) error {
	executable := vendors.Executable(instance.Vendor)
	return vendors.DownloadFile(InstanceDir(instance.ID)+"/"+executable, url)
}

func (instance *Instance) Upgrade(v vendors.Vendor, version string, build int) error {
	status, err := SDManager.Status(instance.ID)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("check status: %s", err.Error())}
	}
	if status.State == "active" {
		return &ValidationError{Message: "server must be stopped before upgrading"}
	}

	downloadURL, err := v.DownloadURL(version, build)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("resolve download URL: %s", err.Error())}
	}
	if err := instance.Download(downloadURL); err != nil {
		return &InternalError{Message: fmt.Sprintf("download: %s", err.Error())}
	}

	instance.Vendor = v.Name()
	instance.Version = version
	instance.Build = build
	if err := WriteInstanceConfig(instance.ID, instance.InstanceConfig); err != nil {
		return &InternalError{Message: fmt.Sprintf("write instance config: %s", err.Error())}
	}
	return nil
}

// EnsureStartReady checks that the instance can start (port availability + memory budget).
func (instance *Instance) EnsureStartReady() error {
	ports, err := instance.ReadPorts()
	if err != nil {
		return err
	}
	if err := checkPortAvailable(ports.Game); err != nil {
		return &ValidationError{Message: fmt.Sprintf("game port: %s", err.Error())}
	}
	if err := checkPortAvailable(ports.RCON); err != nil {
		return &ValidationError{Message: fmt.Sprintf("RCON port: %s", err.Error())}
	}
	if err := CheckMemoryBudget(instance.ID, instance.Memory); err != nil {
		return err
	}
	return nil
}

// Start validates readiness then starts the instance via systemd.
func (instance *Instance) Start() error {
	if err := instance.EnsureStartReady(); err != nil {
		return err
	}
	return SDManager.Start(instance.ID)
}

// Stop stops the instance via systemd.
func (instance *Instance) Stop() error {
	return SDManager.Stop(instance.ID)
}

// Restart restarts the instance via systemd.
func (instance *Instance) Restart() error {
	return SDManager.Restart(instance.ID)
}

// Status returns the current systemd service status.
func (instance *Instance) Status() (*ServiceStatus, error) {
	return SDManager.Status(instance.ID)
}

// Enable enables the instance to start on boot, checking memory budget first.
func (instance *Instance) Enable() error {
	if err := CheckEnableBudget(instance.ID, instance.Memory); err != nil {
		return err
	}
	if err := SDManager.Enable(instance.ID); err != nil {
		return &ValidationError{Message: fmt.Sprintf("enable failed: %s", err.Error())}
	}
	return SDManager.Reload()
}

// Disable disables the instance from starting on boot.
func (instance *Instance) Disable() error {
	if err := SDManager.Disable(instance.ID); err != nil {
		return err
	}
	return SDManager.Reload()
}

// RCON sends a command to the running server via the RCON protocol.
func (instance *Instance) RCON(cmd string) (string, error) {
	ports, err := instance.ReadPorts()
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
	Name         *string  `json:"name"`
	Binary       *string  `json:"binary"`
	JavaArgs     []string `json:"java_args"`
	ServerArgs   []string `json:"server_args"`
	Memory       *int     `json:"memory"`
	GamePort     *int     `json:"game_port"`
	RCONPort     *int     `json:"rcon_port"`
	RCONPassword *string  `json:"rcon_password"`
}

func (instance *Instance) Patch(req PatchRequest) error {
	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Binary != nil {
		instance.Binary = *req.Binary
	}
	if req.JavaArgs != nil {
		instance.JavaArgs = req.JavaArgs
	}
	if req.ServerArgs != nil {
		instance.ServerArgs = req.ServerArgs
	}
	if req.Memory != nil {
		instance.Memory = *req.Memory
	}

	if err := instance.Validate(); err != nil {
		return err
	}
	if err := WriteInstanceConfig(instance.ID, instance.InstanceConfig); err != nil {
		return &InternalError{Message: fmt.Sprintf("write instance config: %s", err.Error())}
	}

	if req.GamePort != nil || req.RCONPort != nil || req.RCONPassword != nil {
		status, err := SDManager.Status(instance.ID)
		if err != nil {
			return &InternalError{Message: fmt.Sprintf("check status: %s", err.Error())}
		}
		if status.State == "active" {
			return &ValidationError{Message: "stop the server before changing ports"}
		}

		ports := instance.Ports
		if req.GamePort != nil {
			ports.Game = *req.GamePort
		}
		if req.RCONPort != nil {
			ports.RCON = *req.RCONPort
		}
		if req.RCONPassword != nil {
			ports.RCONPassword = *req.RCONPassword
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
		if err := instance.WritePorts(ports); err != nil {
			return &InternalError{Message: fmt.Sprintf("write server.properties: %s", err.Error())}
		}
	}

	return nil
}
