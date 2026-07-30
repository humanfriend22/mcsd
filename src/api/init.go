package api

import (
	"fmt"
	"mcsd/core"
	"mcsd/vendors"
	"net/http"
)

type vendorResponse struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type initResponse struct {
	Vendors      []vendorResponse  `json:"vendors"`
	PublicIP     string            `json:"public_ip"`
	LocalIP      string            `json:"local_ip"`
	JavaBinaries []core.JavaBinary `json:"java_binaries"`
}

func getInitial(w http.ResponseWriter, r *http.Request) {
	allVendors := vendors.All
	result := make([]vendorResponse, 0, len(allVendors))
	for _, v := range allVendors {
		versions, err := v.Versions()
		if err != nil {
			writeJSON(w, 500, map[string]string{
				"error": fmt.Sprintf("fetch versions for %s: %v", v.Name(), err),
			})
			return
		}
		result = append(result, vendorResponse{Name: v.Name(), Versions: versions})
	}
	writeJSON(w, 200, initResponse{
		Vendors:      result,
		PublicIP:     cachedPublicIP,
		LocalIP:      cachedLocalIP,
		JavaBinaries: core.DiscoverJavaBinaries(),
	})
}
