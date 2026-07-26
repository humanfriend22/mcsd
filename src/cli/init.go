package cli

import (
	"fmt"

	"mcsd/core"
)

type InitCmd struct {
	Memory int `name:"memory" default:"0" help:"Total RAM (MB) across all instances (default: system RAM - 512)"`
}

func (c *InitCmd) Run() error {
	memory, err := core.Init(c.Memory)
	if err != nil {
		return err
	}

	fmt.Printf("mcsd initialized. memory budget: %d MB\n", memory)
	fmt.Println("\nRun 'sudo mcsd start' to start the HTTP daemon.")
	return nil
}
