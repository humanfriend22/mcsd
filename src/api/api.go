package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"mcsd/core"
)

var (
	staticHandler http.Handler
	sdClient      *core.SDClient
)

func SetStaticFiles(h http.Handler) {
	staticHandler = h
}

func Serve(port int) error {
	if port == 0 {
		port = 8080
	}
	fmt.Printf("Attempting to start mcsd daemon on port %d\n", port)
	var err error
	sdClient, err = core.NewSDClient()
	if err != nil {
		return fmt.Errorf("connect to systemd: %w", err)
	}
	defer sdClient.Close()

	forceRecache()
	go initPublicIP()

	mux := http.NewServeMux()

	// CRUD
	mux.HandleFunc("GET /api/instances/{id}", getInstance)
	mux.HandleFunc("GET /api/instances", listInstances)
	mux.HandleFunc("POST /api/instances", createInstance)
	mux.HandleFunc("PATCH /api/instances/{id}", patchInstance)
	mux.HandleFunc("DELETE /api/instances/{id}", deleteInstance)

	mux.HandleFunc("POST /api/instances/{id}/upgrade", upgradeInstance)

	// systemd
	mux.HandleFunc("POST /api/instances/{id}/start", startInstance)
	mux.HandleFunc("POST /api/instances/{id}/stop", stopInstance)
	mux.HandleFunc("POST /api/instances/{id}/enable", enableInstance)
	mux.HandleFunc("POST /api/instances/{id}/disable", disableInstance)

	// File system
	mux.HandleFunc("GET /api/instances/{id}/files", listFiles)
	mux.HandleFunc("GET /api/instances/{id}/files/content", readFile)
	mux.HandleFunc("POST /api/instances/{id}/files/content", writeFile)
	mux.HandleFunc("DELETE /api/instances/{id}/files", deleteFile)
	mux.HandleFunc("POST /api/instances/{id}/files", uploadFile)

	// Misc
	mux.HandleFunc("POST /api/instances/{id}/rcon", rconInstance)
	mux.HandleFunc("GET /api/instances/{id}/logs", streamLogs)

	// A combined realtime state of instances + host
	mux.HandleFunc("GET /api/state", getAllStates)
	mux.HandleFunc("GET /api/vitals", getVitals)

	// Meant to be called once (rarely changes)
	mux.HandleFunc("GET /api/init", listVendors)

	if staticHandler != nil {
		mux.Handle("/", staticHandler)
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", port), corsMiddleware(mux))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type apiPorts struct {
	Game int `json:"game"`
	RCON int `json:"rcon"`
}

type instanceData struct {
	*core.InstanceConfig
	Ports *apiPorts `json:"ports,omitempty"`
}

func buildInstanceData(id string) (*instanceData, error) {
	config, err := cachedInstanceOrError(id)
	if err != nil {
		return nil, err
	}
	d := &instanceData{InstanceConfig: config}
	if ports, err := config.ReadPorts(); err == nil {
		d.Ports = &apiPorts{Game: ports.Game, RCON: ports.RCON}
	}
	return d, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, os.ErrNotExist) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
