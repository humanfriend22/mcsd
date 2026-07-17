// Single source of truth for supported server vendors
package vendors

type Vendor interface {
	Name() string
	Versions() ([]string, error)
	DownloadURL(version string) (string, error)
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

// Checks if a vendor is valid by name
func IsValid(name string) bool {
	return Get(name) != nil
}

func Executable(id string) string {
	if id == "Bedrock" {
		return "server"
	} else {
		return "server.jar"
	}
}

// Add new vendors here
var All = []Vendor{
	FabricVendor{},
}
