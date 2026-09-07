package core

import (
	"encoding/json"
	"fmt"
	"os"

	"mcsd/vendors"

	. "mcsd/utils"
)

// Master struct for a single server
type Instance struct {
	*InstanceConfig       // config.json fields (flattened via embedding)
	Ports           Ports `json:"ports"` // from server.properties
	InstanceState         // live systemd state (flattened via embedding)
}

type InstanceResult struct {
	*Instance
	Error      error
}

func (ir InstanceResult) MarshalJSON() ([]byte, error) {
	var err string
	if ir.Error != nil {
		err = ir.Error.Error()
	}

	type Alias InstanceResult

	return json.Marshal(&struct {
		Alias
		Error string `json:"error,omitempty"`
	}{
		Alias: Alias(ir),
		Error: err,
	})
}

// LoadInstance builds a fully-populated Instance from config.json, server.properties, and systemd.
func LoadInstance(id string) (*Instance, error) {
	config, err := LoadInstanceConfig(id)
	if err != nil {
		return nil, err
	}

	instance := Instance{
		InstanceConfig: config,
		InstanceState:  loadInstanceState(id),
	}

	ports, err := instance.ReadPorts()
	if err != nil {
		return nil, err
	}
	instance.Ports = ports

	return &instance, nil
}

func ListInstances() ([]InstanceResult, error) {
	ids, err := ListInstanceConfigs()
	if err != nil {
		return nil, err
	}
	instances := make([]InstanceResult, 0)
	for _, id := range ids {
		instance, err := LoadInstance(id)
		instances = append(instances, InstanceResult{Instance: instance, Error: err})
	}
	return instances, nil
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
			_ = DeleteInstanceOverride(instance.ID)
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

	if provisionErr = WriteInstanceOverride(instance.ID, instance.Memory); provisionErr != nil {
		return &InternalError{Message: fmt.Sprintf("write systemd override: %s", provisionErr.Error())}
	}

	return nil
}

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

	if err := DeleteInstanceOverride(id); err != nil {
		return err
	}

	return SDManager.Reload()
}

// Download fetches the server JAR to the instance directory.
func (instance *Instance) Download(url string) error {
	executable := vendors.Executable(instance.Vendor)
	return vendors.DownloadFile(InstanceDir(instance.ID)+"/"+executable, url)
}

func (instance *Instance) Upgrade(vendor vendors.Vendor, version string, build int) error {
	status, err := SDManager.Status(instance.ID)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("check status: %s", err.Error())}
	}
	if status.State == "active" {
		return &ValidationError{Message: "server must be stopped before upgrading"}
	}

	downloadURL, err := vendor.DownloadURL(version, build)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("resolve download URL: %s", err.Error())}
	}
	if err := instance.Download(downloadURL); err != nil {
		return &InternalError{Message: fmt.Sprintf("download: %s", err.Error())}
	}

	instance.Vendor = vendor.Name()
	instance.Version = version
	instance.Build = build
	if err := WriteInstanceConfig(instance.ID, instance.InstanceConfig); err != nil {
		return &InternalError{Message: fmt.Sprintf("write instance config: %s", err.Error())}
	}
	return nil
}

// EnsureStartReady checks that the instance can start (port availability + memory budget).
func (instance *Instance) EnsureStartReady() error {
	if !InstanceOverrideExists(instance.ID) {
		return &ValidationError{Message: "systemd override missing; recreate the instance or reconfigure memory to regenerate it"}
	}

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

// Enable enables the instance to start on boot, checking memory budget first.
func (instance *Instance) Enable() error {
	if !InstanceOverrideExists(instance.ID) {
		return &ValidationError{Message: "systemd override missing; recreate the instance or reconfigure memory to regenerate it"}
	}
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

	if req.Memory != nil {
		if err := WriteInstanceOverride(instance.ID, instance.Memory); err != nil {
			return &InternalError{Message: fmt.Sprintf("write systemd override: %s", err.Error())}
		}
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
