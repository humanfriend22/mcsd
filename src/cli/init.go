package cli

import (
	"fmt"

	"mcsd/core"
)

type InitCmd struct {
	Memory int `name:"memory" default:"0" help:"Total RAM (MB) across all instances (default: system RAM - 1024)"`
}

func (c *InitCmd) Run() error {
	memory := c.Memory
	if memory == 0 {
		total, err := core.TotalSystemMemory()
		if err != nil {
			return fmt.Errorf("read system memory: %w", err)
		}
		memory = total - 1024
	}

	if err := core.Init(memory); err != nil {
		return err
	}

	sdClient, err := core.NewSDClient()
	if err != nil {
		return fmt.Errorf("systemd unavailable: %w", err)
	}
	defer sdClient.Close()
	if err := core.EnableDaemon(sdClient); err != nil {
		return err
	}

	fmt.Printf("mcsd initialized. memory budget: %d MB\n", memory)
	fmt.Println("\nRun 'sudo mcsd start' to start the HTTP daemon.")
	return nil
}
