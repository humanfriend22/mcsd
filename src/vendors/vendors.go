package vendors

import (
	"fmt"

	. "mcsd/utils"
)

type Build struct {
	Number  int    `json:"number"`
	Channel string `json:"channel"`
	Time    string `json:"time"`
}

type Vendor interface {
	Name() string
	Versions() ([]string, error)
	Builds(version string) ([]Build, error)
	DownloadURL(version string, build int) (string, error)
}

var names []string

func Names() []string {
	if len(names) == 0 {
		for _, vendor := range All {
			names = append(names, vendor.Name())
		}
	}
	return names
}

func Get(name string) Vendor {
	for _, vendor := range All {
		if vendor.Name() == name {
			return vendor
		}
	}
	return nil
}

func IsValid(name string) bool {
	return Get(name) != nil
}

func Require(name string) (Vendor, error) {
	v := Get(name)
	if v == nil {
		return nil, &ValidationError{Message: fmt.Sprintf("unknown vendor %q", name)}
	}
	return v, nil
}

func Executable(id string) string {
	if id == "Bedrock" {
		return "server"
	} else {
		return "server.jar"
	}
}

var All = []Vendor{
	FabricVendor{},
	PaperVendor{},
}
