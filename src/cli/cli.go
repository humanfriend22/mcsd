package cli

import (
	"mcsd/core"
	"strings"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Init   InitCmd   `cmd:"" help:"Initialize mcsd on this host"`
	Deinit DeinitCmd `cmd:"" help:"Remove all mcsd service files and data"`

	Instance InstanceCmd `cmd:""`

	// Manage mcsd daemon status
	Start   DaemonStartCmd   `cmd:"" group:"daemon" help:"Start mcsd.service"`
	Stop    DaemonStopCmd    `cmd:"" group:"daemon" help:"Stop mcsd.service"`
	Enable  DaemonEnableCmd  `cmd:"" group:"daemon" help:"Enable mcsd.service (start on boot)"`
	Disable DaemonDisableCmd `cmd:"" group:"daemon" help:"Disable mcsd.service (don't start on boot)"`
	Status  DaemonStatusCmd  `cmd:"" group:"daemon" help:"Show mcsd.service status"`

	// Hidden commands intended to ONLY be run by systemd
	Systemd SystemdCmd `cmd:"" hidden:""`
}

// Manage individual instances
// TODO: create a REPL-like interface to bypass typing the instance id every time
type InstanceCmd struct {
	Create  CreateCmd  `cmd:"" group:"instance" help:"Create a new server instance (interactive)"`
	Edit    EditCmd    `cmd:"" group:"instance" help:"Edit instance config in $EDITOR"`
	Start   StartCmd   `cmd:"" group:"instance" help:"Start a server"`
	Stop    StopCmd    `cmd:"" group:"instance" help:"Stop a server"`
	Restart RestartCmd `cmd:"" group:"instance" help:"Restart a server"`
	Status  StatusCmd  `cmd:"" group:"instance" help:"Show server status (all if no id given)"`
	List    ListCmd    `cmd:"" group:"instance" help:"List all instances"`
	Delete  DeleteCmd  `cmd:"" group:"instance" help:"Delete a server instance and all its data"`
	Enable  EnableCmd  `cmd:"" group:"instance" help:"Enable a server to start on boot"`
	Disable DisableCmd `cmd:"" group:"instance" help:"Disable a server from starting on boot"`
	Rcon    RCONCmd    `cmd:"" name:"rcon" group:"instance" help:"Send an RCON command to a server"`
	Console ConsoleCmd `cmd:"" group:"instance" help:"Stream server logs (journal tail)"`
}

// All cli commands except init and deinit require "init" to be run.
func (c *CLI) BeforeApply(ctx *kong.Context) error {
	cmdString := ctx.Command()

	if !strings.HasPrefix(cmdString, "init") && !strings.HasPrefix(cmdString, "deinit") {
		err := core.EnsureReady()
		return err
	}

	return nil
}

func Execute() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("mcsd"),
		kong.Description("Minecraft server manager"),
		kong.Groups{
			"instance": "Instance commands:",
			"daemon":   "Daemon commands (systmed alias):",
		},
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
