package api

import (
	"io"
	"net"
	"net/http"
	"strings"

	"mcsd/core"
)

type snapshotResponse struct {
	Host   core.HostVitals   `json:"host"`
	Budget core.BudgetVitals `json:"budget"`
}

var cachedPublicIP string

func initPublicIP() {
	resp, err := http.Get("https://api4.ipify.org?format=text")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	cachedPublicIP = strings.TrimSpace(string(ipBytes))
}

var cachedLocalIP string

func initLocalIP() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			cachedLocalIP = ipNet.IP.String()
			return
		}
	}
}

func getVitals(w http.ResponseWriter, r *http.Request) {
	host, err := core.ReadHostVitals()
	if err != nil {
		writeError(w, err)
		return
	}
	budget, err := core.ReadBudgetVitals()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotResponse{Host: host, Budget: budget})
}
