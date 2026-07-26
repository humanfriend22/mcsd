package cli

import (
	"fmt"

	"mcsd/core"
)

type DeinitCmd struct {
	Force bool `short:"f" help:"Skip confirmation prompt"`
}

func (c *DeinitCmd) Run() error {
	if !c.Force {
		fmt.Print("Remove all mcsd service files and data? [y/N] ")
		var resp string
		fmt.Scanln(&resp)
		if resp != "y" && resp != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := core.DeInit(); err != nil {
		return err
	}

	fmt.Println("Removed. Run 'sudo mcsd init' to set up again.")
	return nil
}
