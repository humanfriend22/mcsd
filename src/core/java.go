package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	. "mcsd/utils"
)

type JavaInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Vendor  string `json:"vendor"`
}

func DiscoverJavaBinaries() []JavaInfo {
	seen := make(map[string]bool)
	var results []javaCandidate

	// On Linux/macOS, split PATH. On Windows, split by ';' and also check common locations.
	paths := filepath.SplitList(os.Getenv("PATH"))
	if runtime.GOOS == "windows" {
		for _, dir := range []string{
			filepath.Join(os.Getenv("JAVA_HOME"), "bin"),
			`C:\Program Files\Java`,
			`C:\Program Files (x86)\Java`,
		} {
			if dir != "" && dir != `\bin` {
				paths = append(paths, dir)
			}
		}
	}

	// Also check JAVA_HOME
	if home := os.Getenv("JAVA_HOME"); home != "" {
		binDir := filepath.Join(home, "bin")
		paths = append([]string{binDir}, paths...)
	}

	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			// Match "java", "java-17", "java-21-openjdk", etc. but not "javac", "javadoc", etc.
			if !isJavaBinaryName(name) {
				continue
			}
			abs := filepath.Join(dir, name)
			resolved, err := filepath.EvalSymlinks(abs)
			if err != nil {
				resolved = abs
			}
			if seen[resolved] {
				continue
			}
			seen[resolved] = true
			results = append(results, javaCandidate{path: abs, resolved: resolved})
		}
	}

	var binaries []JavaInfo
	for _, c := range results {
		bin := probeJavaBinary(c.path)
		if bin != nil {
			binaries = append(binaries, *bin)
		}
	}
	return binaries
}

type javaCandidate struct {
	path     string
	resolved string
}

func isJavaBinaryName(name string) bool {
	name = strings.ToLower(name)
	if name == "java" {
		return true
	}
	// Match java-17, java-21, java-21-openjdk, java-17.0.1, etc.
	if strings.HasPrefix(name, "java-") || strings.HasPrefix(name, "java.") {
		// Must not end with common non-executable suffixes
		suffixes := []string{".exe", ".sh", ".bat", ".cmd", ".dll", ".so", ".dylib", ".class", ".jar", ".jnilib"}
		for _, s := range suffixes {
			if strings.HasSuffix(name, s) {
				return false
			}
		}
		return true
	}
	return false
}

func probeJavaBinary(path string) *JavaInfo {
	cmd := exec.Command(path, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	return parseJavaVersion(path, string(out))
}

func parseJavaVersion(path, output string) *JavaInfo {
	// Typical output: openjdk version "21.0.1" 2024-01-16 LTS
	// or: java version "17.0.1" 2021-10-19
	// or: Java(TM) SE Runtime Environment (build 21.0.1+13-LTS-58)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil
	}
	first := lines[0]

	vendor := "Unknown"
	version := ""

	// Parse vendor from first line
	lower := strings.ToLower(first)
	switch {
	case strings.Contains(lower, "openjdk"):
		vendor = "OpenJDK"
	case strings.Contains(lower, "java(TM)"):
		vendor = "Oracle JDK"
	case strings.Contains(lower, "graalvm"):
		vendor = "GraalVM"
	case strings.Contains(lower, "amazon"):
		vendor = "Amazon Corretto"
	case strings.Contains(lower, "temurin") || strings.Contains(lower, "adoptium"):
		vendor = "Eclipse Temurin"
	case strings.Contains(lower, "zulu"):
		vendor = "Azul Zulu"
	case strings.Contains(lower, "dragonwell"):
		vendor = "Alibaba Dragonwell"
	case strings.Contains(lower, "semeru") || strings.Contains(lower, "openj9"):
		vendor = "IBM Semeru"
	}

	// Extract version string between quotes
	if idx := strings.Index(first, `"`); idx >= 0 {
		rest := first[idx+1:]
		if end := strings.Index(rest, `"`); end >= 0 {
			version = rest[:end]
		}
	}

	return &JavaInfo{
		Path:    path,
		Version: version,
		Vendor:  vendor,
	}
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
	return nil
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

func ValidateJavaBinary(path string) error {
	if err := ValidateBinary(path); err != nil {
		return err
	}
	cmd := exec.Command(path, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("not a valid java binary: %s", err.Error())}
	}
	if parseJavaVersion(path, string(out)) == nil {
		return &ValidationError{Message: fmt.Sprintf("could not parse java version output from %s", path)}
	}
	return nil
}
