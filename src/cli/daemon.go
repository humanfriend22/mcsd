package cli

import (
	"fmt"

	"mcsd/core"
)

type DaemonCmd struct {
	Start   DaemonStartCmd   `cmd:"" help:"Start mcsd.service"`
	Stop    DaemonStopCmd    `cmd:"" help:"Stop mcsd.service"`
	Enable  DaemonEnableCmd  `cmd:"" help:"Enable mcsd.service (start on boot)"`
	Disable DaemonDisableCmd `cmd:"" help:"Disable mcsd.service (don't start on boot)"`
	Status  DaemonStatusCmd  `cmd:"" help:"Show mcsd.service status"`
}

type DaemonStartCmd struct{}

func (c *DaemonStartCmd) Run() error {
	return core.SDManager.Start("")
}

type DaemonStopCmd struct{}

func (c *DaemonStopCmd) Run() error {
	return core.SDManager.Stop("")
}

type DaemonEnableCmd struct{}

func (c *DaemonEnableCmd) Run() error {
	if err := core.SDManager.Enable(""); err != nil {
		return err
	}
	fmt.Println("mcsd.service enabled — will start on boot.")
	return nil
}

type DaemonDisableCmd struct{}

func (c *DaemonDisableCmd) Run() error {
	if err := core.SDManager.Disable(""); err != nil {
		return err
	}
	fmt.Println("mcsd.service disabled.")
	return nil
}

type DaemonStatusCmd struct{}

func (c *DaemonStatusCmd) Run() error {
	status, err := core.SDManager.Status("")
	if err != nil {
		return err
	}
	fmt.Printf("%-20s  %-12s  %s\n", "mcsd", status.State, status.SubState)
	return nil
}
