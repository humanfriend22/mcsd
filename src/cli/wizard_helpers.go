package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"mcsd/core"
	"mcsd/vendors"

	"github.com/charmbracelet/huh"
)

// runForm runs a huh form, reporting a user abort separately from a hard error
// so callers don't have to re-derive huh's abort-vs-error distinction each time.
func runForm(form *huh.Form) (aborted bool, err error) {
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func validatePort(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("must be 1-65535")
	}
	return nil
}

func validateRAM(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 512 {
		return fmt.Errorf("must be at least 512")
	}
	return nil
}

// buildJavaArgs returns the JVM heap flags, plus Aikar's flags when flagsIdx == 1.
func buildJavaArgs(ramMB, flagsIdx int) []string {
	args := []string{"-Xms512M", fmt.Sprintf("-Xmx%dM", ramMB)}
	if flagsIdx == 1 {
		args = append(args, core.AikarFlags...)
	}
	return args
}

// hasAikarFlags reports whether args contains Aikar's flags.
func hasAikarFlags(args []string) bool {
	for _, arg := range args {
		if strings.Contains(strings.ToLower(arg), "aikar") {
			return true
		}
	}
	return false
}

// selectVersionAndBuild runs the "pick a version, then pick a build" wizard flow shared by
// instance creation and upgrade. When currentVersion is empty (create), the version step has
// no preset value or note. When it's set (upgrade), a "Current version" note is shown and the
// select defaults to it.
func selectVersionAndBuild(vendor vendors.Vendor, currentVersion string, currentBuild int) (version string, build int, aborted bool, err error) {
	versionList, err := vendor.Versions()
	if err != nil {
		return "", 0, false, fmt.Errorf("fetch versions: %w", err)
	}

	version = currentVersion
	var fields []huh.Field
	if currentVersion != "" {
		fields = append(fields, huh.NewNote().
			Title("Current version").
			Description(fmt.Sprintf("%s %s build %d", vendor.Name(), currentVersion, currentBuild)))
	}
	fields = append(fields, huh.NewSelect[string]().
		Title("Minecraft version").
		Options(buildVersionOptions(versionList)...).
		Value(&version))

	versionGroup := huh.NewGroup(fields...)
	if currentVersion == "" {
		versionGroup = versionGroup.Title("Version")
	}
	if aborted, err := runForm(huh.NewForm(versionGroup)); aborted || err != nil {
		return "", 0, aborted, err
	}

	build = currentBuild
	builds, err := vendor.Builds(version)
	if err != nil {
		return "", 0, false, fmt.Errorf("fetch builds: %w", err)
	}
	if len(builds) > 0 {
		buildForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[int]().
					Title("Server build").
					Options(buildBuildOptions(builds)...).
					Value(&build),
			).Title("Build"),
		)
		if aborted, err := runForm(buildForm); aborted || err != nil {
			return "", 0, aborted, err
		}
	} else if vendor.Name() != "Fabric" {
		fmt.Printf("No builds available for %s %s — using latest available.\n", vendor.Name(), version)
	}

	return version, build, false, nil
}

// networkGroup builds the Network form group (game port, RCON port, RCON password) shared
// by the create and edit wizards.
func networkGroup(gamePortStr, rconPortStr, rconPass *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Game port").
			Value(gamePortStr).
			Validate(validatePort),
		huh.NewInput().
			Title("RCON port").
			Value(rconPortStr).
			Validate(validatePort),
		huh.NewInput().
			Title("RCON password").
			Value(rconPass),
	).Title("Network")
}

// resourcesGroup builds the Resources form group (RAM, plus Java flags for Java vendors)
// shared by the create and edit wizards.
func resourcesGroup(isJava bool, ramStr *string, flagsIdx *int) *huh.Group {
	fields := []huh.Field{
		huh.NewInput().
			Title("RAM limit (MB)").
			Value(ramStr).
			Validate(validateRAM),
	}
	if isJava {
		fields = append(fields, huh.NewSelect[int]().
			Title("Java flags").
			Options(
				huh.NewOption("Aikar's flags (recommended)", 1),
				huh.NewOption("Bare (Xms/Xmx only)", 0),
			).
			Value(flagsIdx))
	}
	return huh.NewGroup(fields...).Title("Resources")
}

func buildVersionOptions(versions []string) []huh.Option[string] {
	options := make([]huh.Option[string], len(versions))
	for i, v := range versions {
		options[i] = huh.NewOption(v, v)
	}
	return options
}

func buildBuildOptions(builds []vendors.Build) []huh.Option[int] {
	options := make([]huh.Option[int], len(builds))
	for i, b := range builds {
		label := fmt.Sprintf("#%d [%s]", b.Number, b.Channel)
		if len(b.Time) >= 10 {
			label += " " + b.Time[:10]
		}
		options[i] = huh.NewOption(label, b.Number)
	}
	return options
}
