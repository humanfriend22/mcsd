package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mcsd/vendors"

	. "mcsd/helpers"
)

// --- Types ---

type InstanceConfig struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Vendor     string   `json:"vendor"`
	Version    string   `json:"version"`
	JavaArgs   []string `json:"java_args"`
	ServerArgs []string `json:"server_args"`
	Memory     int      `json:"memory"`
}

type Ports struct {
	Game         int    `json:"game"`
	RCON         int    `json:"rcon"`
	RCONPassword string `json:"rcon_password,omitempty"`
}

type InstanceState struct {
	*InstanceConfig
	Ports         *Ports `json:"ports,omitempty"`
	State         string `json:"state"`
	Enabled       bool   `json:"enabled"`
	UptimeSeconds int    `json:"uptime_seconds,omitempty"`
	MemoryUsed    int    `json:"memory_used,omitempty"`
}

// --- Private helpers ---

func (instance *InstanceConfig) unit() string {
	return UnitName(instance.ID)
}

func (instance *InstanceConfig) dir() string {
	return InstanceDir(instance.ID)
}

func (instance *InstanceConfig) executable() string {
	return vendors.Executable(instance.Vendor)
}

// --- Validation ---

func (instance *InstanceConfig) Validate() error {
	if !vendors.IsValid(instance.Vendor) {
		return fmt.Errorf("unknown vendor %q", instance.Vendor)
	}
	if instance.Memory < 512 {
		return fmt.Errorf("memory must be at least 512 MB, got %d", instance.Memory)
	}
	return nil
}

func (ports *Ports) Validate() error {
	if ports.Game < 1 || ports.Game > 65535 {
		return fmt.Errorf("game port %d out of range (1-65535)", ports.Game)
	}
	if ports.RCON < 1 || ports.RCON > 65535 {
		return fmt.Errorf("RCON port %d out of range (1-65535)", ports.RCON)
	}
	if ports.Game == ports.RCON {
		return fmt.Errorf("game port and RCON port must be different")
	}
	return nil
}

// --- Persistence ---

func LoadInstanceConfig(id string) (*InstanceConfig, error) {
	path := filepath.Join(InstanceDir(id), "config.json")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("instance %q not found", id)
		}
		return nil, fmt.Errorf("open instance config: %w", err)
	}
	defer f.Close()

	var instance InstanceConfig
	if err := json.NewDecoder(f).Decode(&instance); err != nil {
		return nil, fmt.Errorf("decode instance config: %w", err)
	}
	if instance.ID == "" {
		instance.ID = id
	}
	if instance.Name == "" {
		instance.Name = instance.ID
	}
	return &instance, nil
}

func WriteInstanceConfig(id string, instance *InstanceConfig) error {
	dir := InstanceDir(id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create instance dir: %w", err)
	}
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return fmt.Errorf("encode instance config: %w", err)
	}
	return WriteAtomic(filepath.Join(dir, "config.json"), append(data, '\n'), 0644)
}

func ListInstanceConfigs() ([]string, error) {
	entries, err := os.ReadDir(DefaultBasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read instances dir: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		configPath := filepath.Join(DefaultBasePath, entry.Name(), "config.json")
		if _, err := os.Stat(configPath); err == nil {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func (instance *InstanceConfig) readServerProperties() (map[string]string, error) {
	file, err := os.Open(filepath.Join(instance.dir(), "server.properties"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	props := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			props[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return props, scanner.Err()
}

func (instance *InstanceConfig) ReadPorts() (Ports, error) {
	props, err := instance.readServerProperties()
	if err != nil {
		return Ports{}, fmt.Errorf("read server.properties: %w", err)
	}
	var ports Ports
	if value, ok := props["server-port"]; ok {
		ports.Game, _ = strconv.Atoi(value)
	}
	if value, ok := props["rcon.port"]; ok {
		ports.RCON, _ = strconv.Atoi(value)
	}
	ports.RCONPassword = props["rcon.password"]
	return ports, nil
}

// --- Provisioning ---

func (instance *InstanceConfig) Download(url string) error {
	return vendors.DownloadFile(filepath.Join(instance.dir(), instance.executable()), url)
}

// --- CRUD ---

func (instance *InstanceConfig) Create(ports Ports) error {
	if err := instance.Validate(); err != nil {
		return err
	}
	if err := ports.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(instance.dir(), 0750); err != nil {
		return fmt.Errorf("create instance dir: %w", err)
	}

	// Rollback: if any provisioning step fails, remove the partially-created dir.
	var provisionErr error
	defer func() {
		if provisionErr != nil {
			_ = os.RemoveAll(instance.dir())
		}
	}()

	// Write config.json
	if provisionErr = WriteInstanceConfig(instance.ID, instance); provisionErr != nil {
		return fmt.Errorf("write instance config: %w", provisionErr)
	}

	// Write server.properties
	serverProps := fmt.Sprintf(
		"server-port=%d\nenable-rcon=true\nrcon.port=%d\nrcon.password=%s\nonline-mode=true\n",
		ports.Game, ports.RCON, ports.RCONPassword,
	)
	if provisionErr = WriteAtomic(filepath.Join(instance.dir(), "server.properties"), []byte(serverProps), 0640); provisionErr != nil {
		return fmt.Errorf("write server.properties: %w", provisionErr)
	}

	// Write eula.txt
	if provisionErr = WriteAtomic(filepath.Join(instance.dir(), "eula.txt"), []byte("eula=true\n"), 0640); provisionErr != nil {
		return fmt.Errorf("write eula.txt: %w", provisionErr)
	}

	return nil
}

func (instance *InstanceConfig) Delete(client *SDClient) error {
	return DeleteByID(instance.ID, client)
}

func DeleteByID(id string, client *SDClient) error {
	instance := &InstanceConfig{ID: id}
	unit := instance.unit()

	if status, err := client.Status(unit); err == nil && status.State == "active" {
		if err := client.Stop(unit); err != nil {
			return fmt.Errorf("stop before delete: %w", err)
		}
	}
	if err := client.Disable(unit); err != nil {
		return fmt.Errorf("disable unit: %w", err)
	}
	if _, err := os.Stat(instance.dir()); !os.IsNotExist(err) {
		if err := os.RemoveAll(instance.dir()); err != nil {
			return fmt.Errorf("remove instance dir: %w", err)
		}
	}
	return client.Reload()
}

// --- Lifecycle ---

func (instance *InstanceConfig) Start(client *SDClient) error {
	return client.Start(instance.unit())
}

func (instance *InstanceConfig) Stop(client *SDClient) error {
	return client.Stop(instance.unit())
}

func (instance *InstanceConfig) Restart(client *SDClient) error {
	return client.Restart(instance.unit())
}

func (instance *InstanceConfig) Status(client *SDClient) (*ServiceStatus, error) {
	return client.Status(instance.unit())
}

func (instance *InstanceConfig) Enable(client *SDClient) error {
	if err := client.Enable(instance.unit()); err != nil {
		return err
	}
	return client.Reload()
}

func (instance *InstanceConfig) Disable(client *SDClient) error {
	if err := client.Disable(instance.unit()); err != nil {
		return err
	}
	return client.Reload()
}

func (instance *InstanceConfig) RCON(cmd string) (string, error) {
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

func (instance *InstanceConfig) State(sdClient *SDClient) (*InstanceState, error) {
	unit := UnitName(instance.ID)
	serviceStatus, err := sdClient.Status(unit)
	if err != nil {
		return nil, err
	}

	state := &InstanceState{
		InstanceConfig: instance,
		State:          serviceStatus.State,
		Enabled:        sdClient.IsEnabled(unit),
	}
	if ports, err := instance.ReadPorts(); err == nil {
		ports.RCONPassword = ""
		state.Ports = &ports
	}
	if serviceStatus.State == "active" {
		if since, err := sdClient.ActiveSince(unit); err == nil && !since.IsZero() {
			state.UptimeSeconds = int(time.Since(since).Seconds())
		}
		if bytes, err := sdClient.MemoryUsedBytes(unit); err == nil && bytes > 0 {
			state.MemoryUsed = int(bytes / (1024 * 1024))
		}
	}
	return state, nil
}

// --- Launch (ExecStart entry point called by systemd) ---

func Launch(id string) error {
	if os.Getenv("INVOCATION_ID") == "" {
		fmt.Fprintln(os.Stderr, "mcsd launch must be run by systemd.")
		os.Exit(1)
	}

	if err := ValidateID(id); err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := LoadInstanceConfig(id)
	if err != nil {
		return fmt.Errorf("load instance config: %w", err)
	}

	// Port and budget checks run here — after systemd invokes the process,
	// before exec-ing into the server binary.
	ports, err := instance.ReadPorts()
	if err != nil {
		return fmt.Errorf("read ports: %w", err)
	}
	if err := checkPortAvailable(ports.Game); err != nil {
		return fmt.Errorf("game port: %w", err)
	}
	if err := checkPortAvailable(ports.RCON); err != nil {
		return fmt.Errorf("RCON port: %w", err)
	}

	sdClient, err := NewSDClient()
	if err != nil {
		return fmt.Errorf("connect to systemd: %w", err)
	}
	defer sdClient.Close()

	if err := checkMemoryBudget(sdClient, instance.Memory); err != nil {
		return err
	}

	args, err := resolveArgs(instance)
	if err != nil {
		return fmt.Errorf("resolve command: %w", err)
	}

	if err := os.Chdir(InstanceDir(id)); err != nil {
		return fmt.Errorf("chdir to instance dir: %w", err)
	}

	return syscall.Exec(args[0], args, os.Environ())
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port %d unavailable: %w", port, err)
	}
	ln.Close()
	return nil
}

func checkMemoryBudget(client *SDClient, requestedMemory int) error {
	// Fetch the global memory budget for mcsd
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if config.MemoryBudget <= 0 {
		return nil
	}

	// List all existing services
	units, err := client.List("mcsd-server@*.service")
	if err != nil {
		return fmt.Errorf("budget check: list servers: %w", err)
	}

	// Sum all active instance memory allocations
	var reserved int
	for _, unit := range units {
		if unit.State != "active" {
			continue
		}
		id := UnitToID(unit.Name)
		instance, err := LoadInstanceConfig(id)
		if err != nil {
			continue
		}
		reserved += instance.Memory
	}
	if reserved+requestedMemory > config.MemoryBudget {
		return fmt.Errorf("memory budget exceeded: %dMB reserved + %dMB requested > %dMB budget",
			reserved, requestedMemory, config.MemoryBudget)
	}
	return nil
}

func resolveArgs(instance *InstanceConfig) ([]string, error) {
	if instance.Vendor != "Bedrock" {
		serverArgs := instance.ServerArgs
		if !slices.Contains(serverArgs, "nogui") {
			serverArgs = append(serverArgs, "nogui")
		}
		args := []string{"java"}
		args = append(args, instance.JavaArgs...)
		args = append(args, "-jar", instance.executable())
		args = append(args, serverArgs...)

		// execve requires an absolute path.
		bin, err := exec.LookPath("java")
		if err != nil {
			return nil, fmt.Errorf("java not found in PATH: %w", err)
		}
		args[0] = bin
		return args, nil
	}

	// Native binary (Bedrock) — absolute path because execve doesn't use PATH.
	return append([]string{filepath.Join(instance.dir(), instance.executable())}, instance.ServerArgs...), nil
}
