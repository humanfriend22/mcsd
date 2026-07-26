package cli

import (
	"errors"
	"fmt"
	"strconv"

	"mcsd/core"
	"mcsd/vendors"

	"github.com/charmbracelet/huh"
)

type CreateCmd struct{}

func (c *CreateCmd) Run() error {
	instance, ports, downloadURL, err := runCreateWizard()
	if err != nil {
		return err
	}
	if instance == nil {
		return nil
	}

	if err := instance.Create(ports); err != nil {
		return err
	}
	fmt.Printf("Downloading %s %s…\n", instance.Vendor, instance.Version)
	if downloadURL != "" {
		if err := instance.Download(downloadURL); err != nil {
			return err
		}
	}

	fmt.Printf("\nCreated %q — ready to start.\n", instance.Name)
	return nil
}

func runCreateWizard() (*core.Instance, core.Ports, string, error) {
	zero := core.Ports{}

	var id, name string
	var vendorIdx int

	vendorOptions := make([]huh.Option[int], len(vendors.All))
	for i, v := range vendors.All {
		vendorOptions[i] = huh.NewOption(v.Name(), i)
	}

	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Server ID").
				Value(&id).
				Validate(core.ValidateIDUnique),
			huh.NewInput().
				Title("Display name").
				Value(&name).
				Placeholder("defaults to server ID"),
			huh.NewSelect[int]().
				Title("Server type").
				Options(vendorOptions...).
				Value(&vendorIdx),
		),
	)
	if err := form1.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, zero, "", nil
		}
		return nil, zero, "", err
	}
	if name == "" {
		name = id
	}

	selectedVendor := vendors.All[vendorIdx]
	isJava := selectedVendor.Name() != "Bedrock"

	versionList, err := selectedVendor.Versions()
	if err != nil {
		return nil, zero, "", fmt.Errorf("fetch versions: %w", err)
	}
	versionOptions := make([]huh.Option[string], len(versionList))
	for i, v := range versionList {
		versionOptions[i] = huh.NewOption(v, v)
	}

	// Discover Java binaries (only for Java-based vendors)
	var javaBinaries []core.JavaInfo
	var javaOptions []huh.Option[int]
	javaBinaryIdx := -1
	if isJava {
		javaBinaries = core.DiscoverJavaBinaries()
		javaOptions = make([]huh.Option[int], 0, len(javaBinaries)+1)
		for i, jb := range javaBinaries {
			label := fmt.Sprintf("%s %s — %s", jb.Vendor, jb.Version, jb.Path)
			javaOptions = append(javaOptions, huh.NewOption(label, i))
		}
		javaOptions = append(javaOptions, huh.NewOption("Custom path…", -1))
		if len(javaBinaries) > 0 {
			javaBinaryIdx = 0
		}
	}

	var version, rconPass, customJavaPath string
	var gamePortStr, rconPortStr, ramStr string
	var flagsIdx int

	gamePortStr = "25565"
	rconPortStr = "25575"
	rconPass = "rcon"
	ramStr = strconv.Itoa(core.DefaultMemory())

	// Step 1: Version selection
	versionForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Minecraft version").
				Options(versionOptions...).
				Value(&version),
		).Title("Version"),
	)
	if err := versionForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, zero, "", nil
		}
		return nil, zero, "", err
	}

	// Step 2: Build picker (PaperMC, Purpur, etc.)
	var build int
	builds, err := selectedVendor.Builds(version)
	if err != nil {
		return nil, zero, "", fmt.Errorf("fetch builds: %w", err)
	}
	if len(builds) > 0 {
		buildOptions := make([]huh.Option[int], len(builds))
		for i, b := range builds {
			label := fmt.Sprintf("#%d [%s]", b.Number, b.Channel)
			if len(b.Time) >= 10 {
				label += " " + b.Time[:10]
			}
			buildOptions[i] = huh.NewOption(label, b.Number)
		}
		buildForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[int]().
					Title("Server build").
					Options(buildOptions...).
					Value(&build),
			).Title("Build"),
		)
		if err := buildForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil, zero, "", nil
			}
			return nil, zero, "", err
		}
	} else if selectedVendor.Name() != "Fabric" {
		fmt.Printf("No builds available for %s %s — using latest available.\n", selectedVendor.Name(), version)
	}

	// Step 3: Remaining config (Java, Network, Resources)
	var configGroups []*huh.Group
	if isJava {
		configGroups = []*huh.Group{
			huh.NewGroup(
				huh.NewSelect[int]().
					Title("Java runtime").
					Options(javaOptions...).
					Value(&javaBinaryIdx),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Game port").
					Value(&gamePortStr).
					Validate(validatePort),
				huh.NewInput().
					Title("RCON port").
					Value(&rconPortStr).
					Validate(validatePort),
				huh.NewInput().
					Title("RCON password").
					Value(&rconPass),
			).Title("Network"),
			huh.NewGroup(
				huh.NewInput().
					Title("RAM limit (MB)").
					Value(&ramStr).
					Validate(func(s string) error {
						n, err := strconv.Atoi(s)
						if err != nil || n < 512 {
							return fmt.Errorf("must be at least 512")
						}
						return nil
					}),
				huh.NewSelect[int]().
					Title("Java flags").
					Options(
						huh.NewOption("Aikar's flags (recommended)", 1),
						huh.NewOption("Bare (Xms/Xmx only)", 0),
					).
					Value(&flagsIdx),
			).Title("Resources"),
		}
	} else {
		configGroups = []*huh.Group{
			huh.NewGroup(
				huh.NewInput().
					Title("Game port").
					Value(&gamePortStr).
					Validate(validatePort),
				huh.NewInput().
					Title("RCON port").
					Value(&rconPortStr).
					Validate(validatePort),
				huh.NewInput().
					Title("RCON password").
					Value(&rconPass),
			).Title("Network"),
		}
	}

	configForm := huh.NewForm(configGroups...)
	if err := configForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, zero, "", nil
		}
		return nil, zero, "", err
	}

	// Handle custom Java path
	var javaBin string
	if isJava {
		if javaBinaryIdx == -1 {
			customForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Java binary path").
						Value(&customJavaPath).
						Validate(func(s string) error {
							if s == "" {
								return fmt.Errorf("path is required")
							}
							return core.ValidateJavaBinary(s)
						}),
				),
			)
			if err := customForm.Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					return nil, zero, "", nil
				}
				return nil, zero, "", err
			}
			javaBin = customJavaPath
		} else if len(javaBinaries) > 0 {
			javaBin = javaBinaries[javaBinaryIdx].Path
		}
	}

	gamePort, _ := strconv.Atoi(gamePortStr)
	rconPort, _ := strconv.Atoi(rconPortStr)
	ram, _ := strconv.Atoi(ramStr)

	ports := core.Ports{
		Game:         gamePort,
		RCON:         rconPort,
		RCONPassword: rconPass,
	}
	if err := ports.Validate(); err != nil {
		return nil, zero, "", err
	}

	downloadURL, err := selectedVendor.DownloadURL(version, build)
	if err != nil {
		return nil, zero, "", fmt.Errorf("resolve download URL: %w", err)
	}

	var javaArgs []string
	if isJava {
		javaArgs = []string{"-Xms512M", fmt.Sprintf("-Xmx%dM", ram)}
		if flagsIdx == 1 {
			javaArgs = append(javaArgs, core.AikarFlags...)
		}
	}

	cfg := core.InstanceConfig{
		ID:         id,
		Name:       name,
		Vendor:     selectedVendor.Name(),
		Version:    version,
		Build:      build,
		Binary:     javaBin,
		JavaArgs:   javaArgs,
		ServerArgs: []string{"nogui"},
		Memory:     ram,
	}
	inst, err := core.NewInstance(cfg, ports)
	if err != nil {
		return nil, zero, "", err
	}
	return inst, ports, downloadURL, nil
}

func validatePort(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("must be 1-65535")
	}
	return nil
}
