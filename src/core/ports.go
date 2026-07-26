package core

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "mcsd/utils"
)

// Ports represents the network ports for a Minecraft server instance.
type Ports struct {
	Game         int    `json:"game"`
	RCON         int    `json:"rcon"`
	RCONPassword string `json:"rcon_password"`
}

func (p Ports) Validate() error {
	if p.Game < 1 || p.Game > 65535 {
		return &ValidationError{Message: fmt.Sprintf("game port %d out of range (1-65535)", p.Game)}
	}
	if p.RCON < 1 || p.RCON > 65535 {
		return &ValidationError{Message: fmt.Sprintf("RCON port %d out of range (1-65535)", p.RCON)}
	}
	if p.Game == p.RCON {
		return &ValidationError{Message: "game port and RCON port must be different"}
	}
	return nil
}

// ReadPorts parses server.properties from the given directory.
func ReadPorts(dir string) (Ports, error) {
	file, err := os.Open(filepath.Join(dir, "server.properties"))
	if err != nil {
		return Ports{}, &ServerError{Message: fmt.Sprintf("open server.properties: %s", err.Error())}
	}
	defer file.Close()

	props := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			props[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return Ports{}, &ServerError{Message: fmt.Sprintf("scan server.properties: %s", err.Error())}
	}

	var ports Ports
	if value, ok := props["server-port"]; ok {
		ports.Game, _ = strconv.Atoi(value)
	}
	if value, ok := props["rcon.port"]; ok {
		ports.RCON, _ = strconv.Atoi(value)
	}
	ports.RCONPassword = props["rcon.password"]
	return ports, nil
}

// WriteServerProperties writes server.properties with the given ports.
func WriteServerProperties(dir string, ports Ports) error {
	serverProps := fmt.Sprintf(
		"server-port=%d\nenable-rcon=true\nrcon.port=%d\nrcon.password=%s\nonline-mode=true\n",
		ports.Game, ports.RCON, ports.RCONPassword,
	)
	if err := WriteAtomic(filepath.Join(dir, "server.properties"), []byte(serverProps), 0640); err != nil {
		return &ServerError{Message: fmt.Sprintf("write server.properties: %s", err.Error())}
	}
	return nil
}
