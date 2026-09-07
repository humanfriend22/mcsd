package core

import (
	"fmt"
	"strconv"

	. "mcsd/utils"
)

// MinPort and MaxPort bound valid port numbers for both game and RCON ports.
// CLI wizards reuse these so the client-side check can't drift from this
// authoritative range.
const (
	MinPort = 1
	MaxPort = 65535
)

// Ports represents the network ports for a Minecraft server instance.
type Ports struct {
	Game         int    `json:"game"`
	RCON         int    `json:"rcon"`
	RCONPassword string `json:"rcon_password"`
}

func (ports Ports) Validate() error {
	if ports.Game < MinPort || ports.Game > MaxPort {
		return &ValidationError{Message: fmt.Sprintf("game port %d out of range (%d-%d)", ports.Game, MinPort, MaxPort)}
	}
	if ports.RCON < MinPort || ports.RCON > MaxPort {
		return &ValidationError{Message: fmt.Sprintf("RCON port %d out of range (%d-%d)", ports.RCON, MinPort, MaxPort)}
	}
	if ports.Game == ports.RCON {
		return &ValidationError{Message: "game port and RCON port must be different"}
	}
	return nil
}

// ReadPorts parses server.properties from the given directory.
func (instance Instance) ReadPorts() (Ports, error) {
	props, err := instance.ReadProperties()
	if err != nil {
		return Ports{}, err
	}

	var ports Ports
	if value, ok := props["server-port"]; ok {
		game, err := strconv.Atoi(value)
		if err != nil {
			return Ports{}, &ValidationError{Message: fmt.Sprintf("invalid server-port %q in server.properties: %s", value, err.Error())}
		}
		ports.Game = game
	}
	if value, ok := props["rcon.port"]; ok {
		rcon, err := strconv.Atoi(value)
		if err != nil {
			return Ports{}, &ValidationError{Message: fmt.Sprintf("invalid rcon.port %q in server.properties: %s", value, err.Error())}
		}
		ports.RCON = rcon
	}
	ports.RCONPassword = props["rcon.password"]
	return ports, nil
}

func (instance Instance) WritePorts(ports Ports) error {
	props := Properties{
		"server-port":   strconv.Itoa(ports.Game),
		"enable-rcon":   "true",
		"rcon.port":     strconv.Itoa(ports.RCON),
		"rcon.password": ports.RCONPassword,
		"online-mode":   "true",
	}
	if err := instance.WriteProperties(props, false); err != nil {
		return err
	}
	return nil
}
