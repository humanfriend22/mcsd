package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	. "mcsd/utils"
)

type JavaBinary struct {
	Description string `json:"description"`
	Path    	string `json:"path"`
}

func DiscoverJavaBinaries() []JavaBinary {
	candidates := make([]string, 0)

	// Search the PATH
	path, err := exec.LookPath("java")
	if err == nil {
		candidates = append(candidates, path)
	}

	// TODO:
	// - Add in /usr/libexec/java_home for darwin
	// - Homebrew, SDKMAN, Mise, etc if possible
	// - Local MCSD only installations as well?

	binaries := make([]JavaBinary, 0)

	for _, path := range candidates {
		binary, err := ValidateJavaBinary(path)
		if err != nil {
			fmt.Println(err)
			continue
		}
		binaries = append(binaries, *binary)
	}

	return binaries
}

func ValidateBinary(path string) error {
	if path == "" {
		return &ValidationError{Message: "binary path is empty"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("binary not found: %s", err.Error())}
	}
	if info.IsDir() {
		return &ValidationError{Message: fmt.Sprintf("path is a directory, not a binary: %s", path)}
	}
	if info.Mode()&0111 == 0 {
		return &ValidationError{Message: fmt.Sprintf("binary is not executable: %s", path)}
	}
	return nil
}

func ValidateJavaBinary(path string) (*JavaBinary, error) {
	if err := ValidateBinary(path); err != nil {
		return nil, err
	}

	cmd := exec.Command(path, "-version")
	data, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("error parsing %s:%s\n", path, err)
		return nil, &InternalError{Message: fmt.Sprintf("error parsing %s:%s\n", path, err)}
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	description := string(bytes.TrimSpace(line))
	binary := JavaBinary{Description: description, Path: path}
	return &binary, nil
}

var AikarFlags = []string{
	"-XX:+UseG1GC",
	"-XX:+ParallelRefProcEnabled",
	"-XX:MaxGCPauseMillis=200",
	"-XX:+UnlockExperimentalVMOptions",
	"-XX:+DisableExplicitGC",
	"-XX:+AlwaysPreTouch",
	"-XX:G1NewSizePercent=30",
	"-XX:G1MaxNewSizePercent=40",
	"-XX:G1HeapRegionSize=8M",
	"-XX:G1ReservePercent=20",
	"-XX:G1HeapWastePercent=5",
	"-XX:G1MixedGCCountTarget=4",
	"-XX:InitiatingHeapOccupancyPercent=15",
	"-XX:G1MixedGCLiveThresholdPercent=90",
	"-XX:G1RSetUpdatingPauseTimePercent=5",
	"-XX:SurvivorRatio=32",
	"-XX:+PerfDisableSharedMem",
	"-XX:MaxTenuringThreshold=1",
	"-Dusing.aikars.flags=https://mcflags.emc.gs",
	"-Daikars.new.flags=true",
}
