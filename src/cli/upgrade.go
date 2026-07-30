package cli

import (
	"fmt"

	"mcsd/core"
	"mcsd/vendors"
)

type UpgradeCmd struct {
	ID string `arg:"" help:"Instance ID"`
}

func (c *UpgradeCmd) Run() error {
	inst, err := core.LoadInstance(c.ID)
	if err != nil {
		return err
	}

	status, err := inst.Status()
	if err != nil {
		return err
	}
	if status.State == "active" {
		return fmt.Errorf("instance %q is active — stop it before upgrading", inst.ID)
	}

	vendor, err := vendors.Require(inst.Vendor)
	if err != nil {
		return err
	}

	version, build, aborted, err := selectVersionAndBuild(vendor, inst.Version, inst.Build)
	if aborted || err != nil {
		return err
	}

	if err := inst.Upgrade(vendor, version, build); err != nil {
		return err
	}

	fmt.Printf("Upgraded %s to %s %s build %d.\n", inst.Name, vendor.Name(), version, build)
	return nil
}
