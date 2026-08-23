package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/tabwriter"

	"mcsd/core"
	"mcsd/utils"
)

type StartCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *StartCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	return inst.Start()
}

type StopCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *StopCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	return inst.Stop()
}

type RestartCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *RestartCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	return inst.Restart()
}

type StatusCmd struct {
	ID string `arg:"" optional:"" help:"Instance ID; shows all if omitted"`
}

func (c *StatusCmd) Run() error {
	if c.ID != "" {
		if err := utils.ValidateID(c.ID); err != nil {
			return err
		}
		return proxySystemctl(core.UnitName(c.ID))
	}
	units, err := core.SDManager.List("mcsd-instance@*.service")
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
	instances, err := core.ListInstances()
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		fmt.Println("No servers found.")
		return nil
	}
	return writeInstanceTable(os.Stdout, instances)
}

// writeInstanceTable renders the ID/STATUS table for ListCmd. It writes to
// the given io.Writer rather than os.Stdout directly so the output format is
// testable. A degraded result (Err non-nil) prints the shared degraded-state
// constant as its status word, then — when the error message is non-empty —
// an additional indented line carrying it, immediately after its row.
func writeInstanceTable(w io.Writer, results []core.InstanceResult) error {
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tSTATUS")
	for _, res := range results {
		if res.Err != nil {
			fmt.Fprintf(writer, "%s\t%s\n", res.ID, core.InstanceStateError)
			if msg := res.Err.Error(); msg != "" {
				fmt.Fprintf(writer, "  %s\t\n", msg)
			}
			continue
		}
		fmt.Fprintf(writer, "%s\t%s\n", res.ID, friendlyState(res.Instance.State))
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
	ID    string `arg:"" help:"Instance ID"`
	Force bool   `short:"f" help:"Skip confirmation"`
}

func (c *DeleteCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		// The instance could not be loaded — its config.json or
		// server.properties may be corrupt, or it may not exist at all.
		// Check whether there is anything worth deleting (an on-disk
		// directory or a known systemd unit) before giving up: a
		// corrupt-but-present instance must still be removable (D-04/D-06),
		// and delete is the operator's only remedy once every other
		// lifecycle verb refuses to run against it.
		_, statErr := os.Stat(core.InstanceDir(c.ID))
		dirExists := statErr == nil
		_, statusErr := core.SDManager.Status(c.ID)
		unitKnown := statusErr == nil
		if !dirExists && !unitKnown {
			return err // nothing here either; report the original load error unchanged
		}
		// No inst.Name is available since the config could not be read —
		// identify the instance by its id and say plainly it failed to load.
		if !confirmDelete(fmt.Sprintf("%s (could not be loaded)", c.ID), c.Force) {
			return nil
		}
		return core.DeleteInstance(c.ID)
	}

	if !confirmDelete(inst.Name, c.Force) {
		return nil
	}
	return core.DeleteInstance(inst.ID)
}

// confirmDelete prompts the operator to confirm destroying label's data,
// unless force is set, reusing the accept-on-y-or-Y / "Aborted." semantics
// shared by both DeleteCmd.Run branches. It returns true when the delete
// should proceed.
func confirmDelete(label string, force bool) bool {
	if force {
		return true
	}
	fmt.Printf("Delete %q and all its data? [y/N] ", label)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Aborted.")
		return false
	}
	return true
}

type RCONCmd struct {
	ID      string   `arg:"" help:"Instance ID"`
	Command []string `arg:"" help:"RCON command"`
}

func (c *RCONCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	response, err := inst.RCON(strings.Join(c.Command, " "))
	if err != nil {
		return err
	}
	fmt.Println(response)
	return nil
}

type EnableCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *EnableCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	if err := inst.Enable(); err != nil {
		return err
	}
	fmt.Printf("Enabled %s — will start on boot.\n", c.ID)
	return nil
}

type DisableCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *DisableCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	if err := inst.Disable(); err != nil {
		return err
	}
	fmt.Printf("Disabled %s — will not start on boot.\n", c.ID)
	return nil
}

type ConsoleCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *ConsoleCmd) Run() error {
	inst, err := loadInstance(c.ID)
	if err != nil {
		return err
	}
	return runConsole(inst)
}

func loadInstance(id string) (*core.Instance, error) {
	if err := utils.ValidateID(id); err != nil {
		return nil, err
	}
	return core.LoadInstance(id)
}
