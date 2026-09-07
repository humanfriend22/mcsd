package cli

import (
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

	// ID, Name, Vendor
	var id, name string
	var vendorIdx int

	vendorOptions := make([]huh.Option[int], len(vendors.All))
	for i, v := range vendors.All {
		vendorOptions[i] = huh.NewOption(v.Name(), i)
	}

	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Instance ID").
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
	if aborted, err := runForm(form1); aborted || err != nil {
		return nil, zero, "", err
	}
	if name == "" {
		name = id
	}
	selectedVendor := vendors.All[vendorIdx]
	isJava := selectedVendor.Name() != "Bedrock"

	// Discover Java binaries (only for Java-based vendors)
	var javaBinaries []core.JavaBinary
	var javaOptions []huh.Option[int]
	javaBinaryIdx := -1
	if isJava {
		javaBinaries = core.DiscoverJavaBinaries()
		javaOptions = make([]huh.Option[int], 0, len(javaBinaries)+1)
		for i, b := range javaBinaries {
			label := fmt.Sprintf("%s — %s", b.Description, b.Path)
			javaOptions = append(javaOptions, huh.NewOption(label, i))
		}
		javaOptions = append(javaOptions, huh.NewOption("Custom path…", -1))
		if len(javaBinaries) > 0 {
			javaBinaryIdx = 0
		}
	}

	var rconPass, customJavaPath string
	var gamePortStr, rconPortStr, ramStr string
	var flagsIdx int

	gamePortStr = "25565"
	rconPortStr = "25575"
	rconPass = "rcon"
	ramStr = strconv.Itoa(core.DefaultMemory())

	// Steps 1-2: Version and build selection
	version, build, aborted, err := selectVersionAndBuild(selectedVendor, "", 0)
	if aborted || err != nil {
		return nil, zero, "", err
	}

	// Step 3: Remaining config (Java, Network, Resources)
	var configGroups []*huh.Group
	if isJava {
		configGroups = append(configGroups, huh.NewGroup(
			huh.NewSelect[int]().
				Title("Java runtime").
				Options(javaOptions...).
				Value(&javaBinaryIdx),
		))
	}
	configGroups = append(configGroups,
		networkGroup(&gamePortStr, &rconPortStr, &rconPass),
		resourcesGroup(isJava, &ramStr, &flagsIdx),
	)

	configForm := huh.NewForm(configGroups...)
	if aborted, err := runForm(configForm); aborted || err != nil {
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
							_, err := core.ValidateJavaBinary(s)
							return err
						}),
				),
			)
			if aborted, err := runForm(customForm); aborted || err != nil {
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
		javaArgs = buildJavaArgs(ram, flagsIdx)
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
	inst := &core.Instance{InstanceConfig: &cfg, Ports: ports}
	if err := inst.Validate(); err != nil {
		return nil, zero, "", err
	}
	return inst, ports, downloadURL, nil
}
