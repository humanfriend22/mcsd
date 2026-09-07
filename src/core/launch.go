package core

import (
	"fmt"
	"net"
	"os"
	"os/exec"
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

	instance, err := LoadInstance(id)
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("load instance config: %s", err.Error())}
	}

	if err := instance.EnsureStartReady(); err != nil {
		return err
	}

	args, err := instance.resolveArgs()
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("resolve args: %s", err.Error())}
	}

	if err := os.Chdir(InstanceDir(id)); err != nil {
		return &InternalError{Message: fmt.Sprintf("chdir to instance dir: %s", err.Error())}
	}

	return syscall.Exec(args[0], args, os.Environ())
}

func (instance *Instance) resolveArgs() ([]string, error) {
	if instance.Binary == "" {
		return nil, &InternalError{Message: fmt.Sprintf("binary not configured (%s)", instance.Binary)}
	}
	binary, err := exec.LookPath(instance.Binary)
	if err != nil {
		return nil, &InternalError{Message: fmt.Sprintf("binary not found (%s): %s", instance.Binary, err.Error())}
	}

	if len(instance.JavaArgs) > 0 {
		serverArgs := instance.ServerArgs
		if !slices.Contains(serverArgs, "nogui") {
			serverArgs = append(serverArgs, "nogui")
		}

		args := []string{binary}
		args = append(args, instance.JavaArgs...)
		executable := vendors.Executable(instance.Vendor)
		args = append(args, "-jar", executable)
		args = append(args, serverArgs...)

		return args, nil
	}

	return append([]string{binary}, instance.ServerArgs...), nil
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("port %d unavailable: %s", port, err.Error())}
	}
	ln.Close()
	return nil
}
