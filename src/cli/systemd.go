package cli

import (
	"fmt"

	"mcsd/api"
	"mcsd/core"
)

// SystemdCmd groups commands invoked by systemd via ExecStart/ExecStop — not for direct use.
type SystemdCmd struct {
	Launch SystemdLaunchCmd `cmd:"" help:"ExecStart= handler"`
	Stop   SystemdStopCmd   `cmd:"" help:"ExecStop= handler"`
	Serve  SystemdServeCmd  `cmd:"" help:"HTTP daemon (Phase 5)"`
}

type SystemdLaunchCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *SystemdLaunchCmd) Run() error {
	return core.Launch(c.ID)
}

type SystemdStopCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *SystemdStopCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	resp, err := inst.RCON("stop")
	if err != nil {
		// Non-fatal: systemd will SIGKILL after TimeoutStopSec anyway.
		fmt.Printf("RCON stop failed (server may already be down): %v\n", err)
		return nil
	}
	fmt.Println(resp)
	return nil
}

type SystemdServeCmd struct{}

func (c *SystemdServeCmd) Run() error {
	config, err := core.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return api.Serve(config.Port)
}
