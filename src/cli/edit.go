package cli

import (
	"fmt"
	"strconv"

	"mcsd/core"

	"github.com/charmbracelet/huh"
)

type EditCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *EditCmd) Run() error {
	inst, err := core.LoadInstance(c.ID)
	if err != nil {
		return err
	}

	status, err := inst.Status()
	if err != nil {
		return err
	}
	if status.State == "active" {
		return fmt.Errorf("instance %q is active — stop it before editing", inst.ID)
	}

	isJava := inst.Vendor != "Bedrock"

	name := inst.Name
	gamePortStr := strconv.Itoa(inst.Ports.Game)
	rconPortStr := strconv.Itoa(inst.Ports.RCON)
	rconPass := inst.Ports.RCONPassword
	ramStr := strconv.Itoa(inst.Memory)
	flagsIdx := 0
	if hasAikarFlags(inst.JavaArgs) {
		flagsIdx = 1
	}

	summary := fmt.Sprintf(
		"ID: %s\nVendor: %s\nVersion: %s\nBuild: %d\nBinary: %s\n\nUse 'mcsd upgrade %s' to change vendor/version/build.",
		inst.ID, inst.Vendor, inst.Version, inst.Build, inst.Binary, inst.ID,
	)

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewNote().
				Title("Current configuration").
				Description(summary),
			huh.NewInput().
				Title("Display name").
				Value(&name),
		),
		networkGroup(&gamePortStr, &rconPortStr, &rconPass),
		resourcesGroup(isJava, &ramStr, &flagsIdx),
	}

	form := huh.NewForm(groups...)
	if aborted, err := runForm(form); aborted || err != nil {
		return err
	}

	gamePort, _ := strconv.Atoi(gamePortStr)
	rconPort, _ := strconv.Atoi(rconPortStr)
	ram, _ := strconv.Atoi(ramStr)

	req := core.PatchRequest{
		Name:         &name,
		Memory:       &ram,
		GamePort:     &gamePort,
		RCONPort:     &rconPort,
		RCONPassword: &rconPass,
	}
	if isJava {
		req.JavaArgs = buildJavaArgs(ram, flagsIdx)
	}

	if err := inst.Patch(req); err != nil {
		return err
	}

	fmt.Printf("Updated %s.\n", inst.Name)
	return nil
}
