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
	var err error
	sdClient, err = core.NewSDClient()
	if err != nil {
		return fmt.Errorf("connect to systemd: %w", err)
	}
	defer sdClient.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/instances", listInstances)
	mux.HandleFunc("POST /api/instances", createInstance)
	mux.HandleFunc("GET /api/instances/{id}", getInstance)
	mux.HandleFunc("PATCH /api/instances/{id}", patchInstance)
	mux.HandleFunc("DELETE /api/instances/{id}", deleteInstance)
	mux.HandleFunc("POST /api/instances/{id}/start", startInstance)
	mux.HandleFunc("POST /api/instances/{id}/stop", stopInstance)
	mux.HandleFunc("POST /api/instances/{id}/enable", enableInstance)
	mux.HandleFunc("POST /api/instances/{id}/disable", disableInstance)
	mux.HandleFunc("POST /api/instances/{id}/upgrade", upgradeInstance)
	mux.HandleFunc("POST /api/instances/{id}/rcon", rconInstance)
	mux.HandleFunc("GET /api/instances/{id}/logs", streamLogs)
	mux.HandleFunc("GET /api/instances/{id}/files", listFiles)
	mux.HandleFunc("GET /api/instances/{id}/files/content", readFile)
	mux.HandleFunc("POST /api/instances/{id}/files/content", writeFile)
	mux.HandleFunc("DELETE /api/instances/{id}/files", deleteFile)
	mux.HandleFunc("POST /api/instances/{id}/files", uploadFile)
	mux.HandleFunc("GET /api/vendors", listVendors)
	mux.HandleFunc("GET /api/vitals", getVitals)
	mux.HandleFunc("GET /api/public-ip", getPublicIP)
	if staticHandler != nil {
		mux.Handle("/", staticHandler)
	}
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func buildInstanceResponse(id string) (*core.InstanceState, error) {
	instance, err := core.LoadInstanceConfig(id)
	if err != nil {
		return nil, err
	}
	return instance.State(sdClient)
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
