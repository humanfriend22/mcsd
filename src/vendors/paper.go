package vendors

import (
	"fmt"
	"sort"
	"strings"
)

const paperUA = "mcsd/0.1 (https://github.com/anomalyco/mcsd)"

type PaperVendor struct{}

func (PaperVendor) Name() string { return "Paper" }

func (PaperVendor) Versions() ([]string, error) {
	var resp struct {
		Versions map[string][]string `json:"versions"`
	}
	if err := getJSONWithHeaders("https://fill.papermc.io/v3/projects/paper", map[string]string{"User-Agent": paperUA}, &resp); err != nil {
		return nil, err
	}

	var out []string
	keys := make([]string, 0, len(resp.Versions))
	for k := range resp.Versions {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareVersions(keys[i], keys[j]) > 0
	})
	for _, k := range keys {
		out = append(out, resp.Versions[k]...)
	}
	return out, nil
}

func (PaperVendor) Builds(version string) ([]Build, error) {
	var builds []struct {
		ID      int    `json:"id"`
		Channel string `json:"channel"`
		Time    string `json:"time"`
	}
	url := fmt.Sprintf("https://fill.papermc.io/v3/projects/paper/versions/%s/builds", version)
	if err := getJSONWithHeaders(url, map[string]string{"User-Agent": paperUA}, &builds); err != nil {
		return nil, err
	}
	out := make([]Build, len(builds))
	for i, b := range builds {
		out[i] = Build{Number: b.ID, Channel: b.Channel, Time: b.Time}
	}
	return out, nil
}

func (PaperVendor) DownloadURL(version string, build int) (string, error) {
	var resp struct {
		Downloads map[string]struct {
			URL string `json:"url"`
		} `json:"downloads"`
	}
	url := fmt.Sprintf("https://fill.papermc.io/v3/projects/paper/versions/%s/builds/%d", version, build)
	if build == 0 {
		url = fmt.Sprintf("https://fill.papermc.io/v3/projects/paper/versions/%s/builds/latest", version)
	}
	if err := getJSONWithHeaders(url, map[string]string{"User-Agent": paperUA}, &resp); err != nil {
		return "", err
	}
	dl, ok := resp.Downloads["server:default"]
	if !ok {
		return "", fmt.Errorf("no server:default download for paper %s build %d", version, build)
	}
	return dl.URL, nil
}

func compareVersions(a, b string) int {
	ap, bp := parseParts(a), parseParts(b)
	n := min(len(ap), len(bp))
	for i := 0; i < n; i++ {
		if ap[i] != bp[i] {
			return ap[i] - bp[i]
		}
	}
	return len(ap) - len(bp)
}

func parseParts(v string) []int {
	var parts []int
	for _, p := range strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-'
	}) {
		var n int
		fmt.Sscanf(p, "%d", &n)
		parts = append(parts, n)
	}
	return parts
}
