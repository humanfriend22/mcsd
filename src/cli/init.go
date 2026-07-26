package cli

import (
	"fmt"

	"mcsd/core"
)

type InitCmd struct {
	Memory int `name:"memory" default:"0" help:"Total RAM (MB) across all instances (default: system RAM - 512)"`
}

func (c *InitCmd) Run() error {
	memory := c.Memory
	if memory == 0 {
		total, err := core.TotalSystemMemory()
		if err != nil {
			return fmt.Errorf("read system memory: %w", err)
		}
		memory = total - 512
	}

	if err := core.Init(memory); err != nil {
		return err
	}

	if err := core.SDManager.Enable(""); err != nil {
		return err
	}

	fmt.Printf("mcsd initialized. memory budget: %d MB\n", memory)
	fmt.Println("\nRun 'sudo mcsd start' to start the HTTP daemon.")
	return nil
}
