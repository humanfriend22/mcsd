package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/tabwriter"

	"mcsd/core"
	"mcsd/helpers"
)

type StartCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *StartCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()
	return instance.Start(sdClient)
}

type StopCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *StopCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()
	return instance.Stop(sdClient)
}

type RestartCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *RestartCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()
	return instance.Restart(sdClient)
}

type StatusCmd struct {
	ID string `arg:"" optional:"" help:"InstanceConfig ID; shows all if omitted"`
}

func (c *StatusCmd) Run() error {
	if c.ID != "" {
		if err := helpers.ValidateID(c.ID); err != nil {
			return err
		}
		return proxySystemctl(core.UnitName(c.ID))
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	units, err := sdClient.List("mcsd-server@*.service")
	sdClient.Close()
	if err != nil {
		return err
	}
	if len(units) == 0 {
		fmt.Println("No servers found.")
		return nil
	}
	names := make([]string, len(units))
	for i, unit := range units {
		names[i] = unit.Name
	}
	return proxySystemctl(names...)
}

func proxySystemctl(units ...string) error {
	path, err := exec.LookPath("systemctl")
	if err != nil {
		return fmt.Errorf("systemctl not found: %w", err)
	}
	args := append([]string{"systemctl", "status"}, units...)
	return syscall.Exec(path, args, os.Environ())
}

type ListCmd struct{}

func (c *ListCmd) Run() error {
	ids, err := core.ListInstanceConfigs()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		fmt.Println("No servers found.")
		return nil
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tSTATUS")
	for _, id := range ids {
		status, err := sdClient.Status(core.UnitName(id))
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "%s\t%s\n", id, friendlyState(status.State))
	}
	return writer.Flush()
}

func friendlyState(state string) string {
	switch state {
	case "active":
		return "running"
	case "inactive":
		return "stopped"
	case "activating":
		return "starting"
	case "deactivating":
		return "stopping"
	default:
		return state
	}
}

type DeleteCmd struct {
	ID    string `arg:"" help:"InstanceConfig ID"`
	Force bool   `short:"f" help:"Skip confirmation"`
}

func (c *DeleteCmd) Run() error {
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()

	instance, err := loadInstance(c.ID)
	if err != nil {
		// Config missing — check for an orphaned systemd unit before giving up.
		unit := core.UnitName(c.ID)
		if _, statusErr := sdClient.Status(unit); statusErr != nil {
			return err // nothing in systemd either; report original error
		}
		return core.DeleteByID(c.ID, sdClient)
	}

	if !c.Force {
		fmt.Printf("Delete %q and all its data? [Y/n] ", instance.Name)
		var response string
		fmt.Scanln(&response)
		if response == "n" || response == "N" {
			fmt.Println("Aborted.")
			return nil
		}
	}
	return instance.Delete(sdClient)
}

type RCONCmd struct {
	ID      string   `arg:"" help:"InstanceConfig ID"`
	Command []string `arg:"" help:"RCON command"`
}

func (c *RCONCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	response, err := instance.RCON(strings.Join(c.Command, " "))
	if err != nil {
		return err
	}
	fmt.Println(response)
	return nil
}

type EnableCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *EnableCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()
	if err := instance.Enable(sdClient); err != nil {
		return err
	}
	fmt.Printf("Enabled %s — will start on boot.\n", c.ID)
	return nil
}

type DisableCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *DisableCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	sdClient, err := core.NewSDClient()
	if err != nil {
		return err
	}
	defer sdClient.Close()
	if err := instance.Disable(sdClient); err != nil {
		return err
	}
	fmt.Printf("Disabled %s — will not start on boot.\n", c.ID)
	return nil
}

type ConsoleCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *ConsoleCmd) Run() error {
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	return runConsole(instance)
}

func loadInstance(id string) (*core.InstanceConfig, error) {
	if err := helpers.ValidateID(id); err != nil {
		return nil, err
	}
	return core.LoadInstanceConfig(id)
}
