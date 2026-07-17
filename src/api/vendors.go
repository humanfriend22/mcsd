package api

import (
	"fmt"
	"mcsd/vendors"
	"net/http"
)

type vendorResponse struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type initResponse struct {
	Vendors  []vendorResponse `json:"vendors"`
	PublicIP string           `json:"public_ip"`
}

func listVendors(w http.ResponseWriter, r *http.Request) {
	recache()

	result := make([]vendorResponse, 0, len(vendors.All))
	for _, v := range vendors.All {
		versions, err := v.Versions()
		if err != nil {
			writeJSON(w, 500, map[string]string{
				"error": fmt.Sprintf("fetch versions for %s: %v", v.Name(), err),
			})
			return
		}
		result = append(result, vendorResponse{Name: v.Name(), Versions: versions})
	}
	writeJSON(w, 200, initResponse{Vendors: result, PublicIP: cachedPublicIP})
}
