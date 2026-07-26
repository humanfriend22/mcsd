package vendors

import "fmt"

type FabricVendor struct{}

func (FabricVendor) Name() string { return "Fabric" }

func (FabricVendor) Versions() ([]string, error) {
	return fabricStableVersions("https://meta.fabricmc.net/v2/versions/game/intermediary")
}

func (FabricVendor) Builds(version string) ([]Build, error) {
	return nil, nil
}

func (FabricVendor) DownloadURL(version string, build int) (string, error) {
	loader, err := fabricLatestStable("https://meta.fabricmc.net/v2/versions/loader")
	if err != nil {
		return "", fmt.Errorf("fetch loader version: %w", err)
	}
	installer, err := fabricLatestStable("https://meta.fabricmc.net/v2/versions/installer")
	if err != nil {
		return "", fmt.Errorf("fetch installer version: %w", err)
	}
	return fmt.Sprintf(
		"https://meta.fabricmc.net/v2/versions/loader/%s/%s/%s/server/jar",
		version, loader, installer,
	), nil
}

func fabricLatestStable(apiURL string) (string, error) {
	versions, err := fabricStableVersions(apiURL)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no stable versions at %s", apiURL)
	}
	return versions[0], nil
}

func fabricStableVersions(apiURL string) ([]string, error) {
	var items []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := getJSON(apiURL, &items); err != nil {
		return nil, err
	}
	var versions []string
	for _, item := range items {
		if item.Stable {
			versions = append(versions, item.Version)
		}
	}
	return versions, nil
}
