package core

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"

	"mcsd/vendors"

	. "mcsd/utils"
)

// Launch is the ExecStart entry point called by systemd for each instance.
func Launch(id string) error {
	if os.Getenv("INVOCATION_ID") == "" {
		fmt.Fprintln(os.Stderr, "mcsd launch must be run by systemd.")
		os.Exit(1)
	}

	if err := ValidateID(id); err != nil {
		return &ValidationError{Message: fmt.Sprintf("invalid instance ID: %s", err.Error())}
	}

	inst, err := LoadInstance(id)
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("load instance config: %s", err.Error())}
	}

	if err := inst.EnsureStartReady(); err != nil {
		return err
	}

	args, err := resolveArgs(inst)
	if err != nil {
		return &ServerError{Message: fmt.Sprintf("resolve command: %s", err.Error())}
	}

	if err := os.Chdir(InstanceDir(id)); err != nil {
		return &ServerError{Message: fmt.Sprintf("chdir to instance dir: %s", err.Error())}
	}

	return syscall.Exec(args[0], args, os.Environ())
}

func resolveArgs(inst *Instance) ([]string, error) {
	if len(inst.JavaArgs) > 0 {
		javaBin := inst.Binary
		if javaBin == "" {
			javaBin = "java"
		}
		bin, err := exec.LookPath(javaBin)
		if err != nil {
			return nil, &ServerError{Message: fmt.Sprintf("java not found: %s", err.Error())}
		}
		serverArgs := inst.ServerArgs
		if !slices.Contains(serverArgs, "nogui") {
			serverArgs = append(serverArgs, "nogui")
		}
		args := []string{bin}
		args = append(args, inst.JavaArgs...)
		executable := vendors.Executable(inst.Vendor)
		args = append(args, "-jar", executable)
		args = append(args, serverArgs...)
		return args, nil
	}

	// Bedrock mode — use configured binary or default to "server" in instance dir.
	bin := inst.Binary
	if bin == "" {
		bin = filepath.Join(InstanceDir(inst.ID), "server")
	}
	return append([]string{bin}, inst.ServerArgs...), nil
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("port %d unavailable: %s", port, err.Error())}
	}
	ln.Close()
	return nil
}
