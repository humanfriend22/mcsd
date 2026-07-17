package core

import "fmt"

const daemonUnit = "mcsd.service"

func EnableDaemon(client *SDClient) error {
	if err := client.Enable(daemonUnit); err != nil {
		return fmt.Errorf("enable %s: %w", daemonUnit, err)
	}
	return client.Reload()
}

func DisableDaemon(client *SDClient) error {
	if err := client.Disable(daemonUnit); err != nil {
		return fmt.Errorf("disable %s: %w", daemonUnit, err)
	}
	return client.Reload()
}

func StartDaemon(client *SDClient) error {
	return client.Start(daemonUnit)
}

func StopDaemon(client *SDClient) error {
	return client.Stop(daemonUnit)
}

func StatusDaemon(client *SDClient) (*ServiceStatus, error) {
	return client.Status(daemonUnit)
}
