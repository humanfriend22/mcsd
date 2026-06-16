package cli

import (
	"fmt"
	"strconv"

	"mcsd/core"
	"mcsd/helpers"
	"mcsd/vendors"

	"github.com/manifoldco/promptui"
)

type CreateCmd struct{}

func (c *CreateCmd) Run() error {
	if err := core.EnsureReady(); err != nil {
		return err
	}

	instance, ports, downloadURL, err := runCreateWizard()
	if err != nil {
		return err
	}
	if instance == nil {
		return nil // cancelled
	}

	if err := instance.Create(ports); err != nil {
		return err
	}
	if downloadURL != "" {
		if err := instance.Download(downloadURL); err != nil {
			return err
		}
	}

	fmt.Printf("\nCreated %q — ready to start.\n", instance.Name)
	return nil
}

func runCreateWizard() (*core.InstanceConfig, core.Ports, string, error) {
	zero := core.Ports{}

	id, cancelled, err := promptText("Server ID", "", helpers.ValidateID)
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	name, cancelled, err := promptText("Display name", id, nil)
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	typeLabels := make([]string, len(vendors.All))
	for i, v := range vendors.All {
		typeLabels[i] = v.Name()
	}
	typeIdx, cancelled, err := promptSelect("Server type", typeLabels)
	if cancelled || err != nil {
		return nil, zero, "", err
	}
	selectedVendor := vendors.All[typeIdx]

	versionList, fetchErr := selectedVendor.Versions()
	if fetchErr != nil {
		return nil, zero, "", fmt.Errorf("fetch versions: %w", fetchErr)
	}
	versionIdx, cancelled, err := promptSelect("Minecraft version", versionList)
	if cancelled || err != nil {
		return nil, zero, "", err
	}
	version := versionList[versionIdx]

	gamePort, cancelled, err := promptInt("Game port", 25565, validatePort)
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	rconPort, cancelled, err := promptInt("RCON port", 25575, validatePort)
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	rconPass, cancelled, err := promptPassword("RCON password", "rcon")
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	ram, cancelled, err := promptInt("RAM limit (MB)", 2048, func(s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < 512 {
			return fmt.Errorf("must be at least 512")
		}
		return nil
	})
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	flagsIdx, cancelled, err := promptSelect("Java flags", []string{"Bare (Xms/Xmx only)", "Aikar's flags (recommended)"})
	if cancelled || err != nil {
		return nil, zero, "", err
	}

	ports := core.Ports{
		Game:         gamePort,
		RCON:         rconPort,
		RCONPassword: rconPass,
	}
	if err := ports.Validate(); err != nil {
		return nil, zero, "", err
	}

	javaArgs := []string{"-Xms512M", fmt.Sprintf("-Xmx%dM", ram)}
	if flagsIdx == 1 {
		javaArgs = append(javaArgs, aikarFlags...)
	}

	downloadURL, err := selectedVendor.DownloadURL(version)
	if err != nil {
		return nil, zero, "", fmt.Errorf("resolve download URL: %w", err)
	}
	if downloadURL != "" {
		fmt.Printf("Downloading %s...\n", selectedVendor.Name())
	}

	return &core.InstanceConfig{
		ID:         id,
		Name:       name,
		Vendor:     selectedVendor.Name(),
		Version:    version,
		JavaArgs:   javaArgs,
		ServerArgs: []string{"nogui"},
		Memory:     ram,
	}, ports, downloadURL, nil
}

var aikarFlags = []string{
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

var promptTpls = &promptui.PromptTemplates{
	Prompt:  fmt.Sprintf(`{{ "%s" | bold }} {{ . | bold }}: `, promptui.IconInitial),
	Valid:   fmt.Sprintf(`{{ "%s" | green }} {{ . | bold }}: `, promptui.IconGood),
	Invalid: fmt.Sprintf(`{{ "%s" | red }} {{ . | bold }}: `, promptui.IconBad),
	Success: fmt.Sprintf(`{{ "%s" | green }} `, promptui.IconGood),
}

func promptText(label, defaultVal string, validate promptui.ValidateFunc) (string, bool, error) {
	var wrapped promptui.ValidateFunc
	if validate != nil {
		wrapped = func(s string) error {
			if s == "" && defaultVal != "" {
				return nil
			}
			return validate(s)
		}
	}

	p := promptui.Prompt{
		Label:     label,
		Default:   defaultVal,
		AllowEdit: true,
		Validate:  wrapped,
		Templates: promptTpls,
	}
	val, err := p.Run()
	if isCancel(err) {
		return "", true, nil
	}
	if err != nil {
		return val, false, err
	}
	if val == "" {
		return defaultVal, false, nil
	}
	return val, false, nil
}

func promptSelect(label string, items []string) (int, bool, error) {
	s := promptui.Select{
		Label: label,
		Items: items,
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ . }}`, promptui.IconGood),
		},
	}
	idx, _, err := s.Run()
	if isCancel(err) {
		return 0, true, nil
	}
	return idx, false, err
}

func promptInt(label string, defaultVal int, validate promptui.ValidateFunc) (int, bool, error) {
	val, cancelled, err := promptText(label, strconv.Itoa(defaultVal), validate)
	if cancelled || err != nil {
		return 0, cancelled, err
	}
	n, _ := strconv.Atoi(val)
	return n, false, nil
}

func promptPassword(label, defaultVal string) (string, bool, error) {
	p := promptui.Prompt{
		Label:     label,
		Default:   defaultVal,
		AllowEdit: true,
		Mask:      '*',
		Templates: promptTpls,
	}
	val, err := p.Run()
	if isCancel(err) {
		return "", true, nil
	}
	if err != nil {
		return val, false, err
	}
	if val == "" {
		return defaultVal, false, nil
	}
	return val, false, nil
}

func validatePort(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("must be 1–65535")
	}
	return nil
}

func isCancel(err error) bool {
	return err == promptui.ErrInterrupt || err == promptui.ErrAbort
}
