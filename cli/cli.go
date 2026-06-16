package cli

import "github.com/alecthomas/kong"

type CLI struct {
	Create  CreateCmd  `cmd:"" group:"server" help:"Create a new server instance (interactive)"`
	Edit    EditCmd    `cmd:"" group:"server" help:"Edit instance config in $EDITOR"`
	Start   StartCmd   `cmd:"" group:"server" help:"Start a server"`
	Stop    StopCmd    `cmd:"" group:"server" help:"Stop a server"`
	Restart RestartCmd `cmd:"" group:"server" help:"Restart a server"`
	Status  StatusCmd  `cmd:"" group:"server" help:"Show server status (all if no id given)"`
	List    ListCmd    `cmd:"" group:"server" help:"List all instances"`
	Delete  DeleteCmd  `cmd:"" group:"server" help:"Delete a server instance and all its data"`
	Enable  EnableCmd  `cmd:"" group:"server" help:"Enable a server to start on boot"`
	Disable DisableCmd `cmd:"" group:"server" help:"Disable a server from starting on boot"`
	Rcon    RCONCmd    `cmd:"" name:"rcon" group:"server" help:"Send an RCON command to a server"`
	Console ConsoleCmd `cmd:"" group:"server" help:"Stream server logs (journal tail)"`

	Init   InitCmd   `cmd:"" group:"admin" help:"Initialize mcsd on this host"`
	Deinit DeinitCmd `cmd:"" group:"admin" help:"Remove all mcsd service files and data"`
	Daemon DaemonCmd `cmd:"" group:"admin" help:"Manage the mcsd HTTP daemon (mcsd.service)"`

	Systemd SystemdCmd `cmd:"" hidden:""`
}

func Execute() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("mcsd"),
		kong.Description("Minecraft server manager"),
		kong.Groups{
			"server": "Server commands:",
			"admin":  "Admin commands:",
		},
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
