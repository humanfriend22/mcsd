package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"mcsd/core"
	"mcsd/vendors"
)

func listInstances(w http.ResponseWriter, r *http.Request) {
	ids := getCachedIDs()
	results := make([]*instanceData, 0, len(ids))
	for _, id := range ids {
		resp, err := buildInstanceData(id)
		if err != nil {
			continue
		}
		results = append(results, resp)
	}
	writeJSON(w, http.StatusOK, results)
}

func getInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp, err := buildInstanceData(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type createRequest struct {
	core.InstanceConfig
	Ports core.Ports `json:"ports"`
}

func createInstance(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	vendor := vendors.Get(req.Vendor)
	if vendor == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown vendor %q", req.Vendor)})
		return
	}

	downloadURL, err := vendor.DownloadURL(req.Version)
	if err != nil {
		writeError(w, err)
		return
	}

	instance := req.InstanceConfig
	if err := instance.Create(req.Ports); err != nil {
		writeError(w, err)
		return
	}
	setCachedInstance(req.ID, &instance)
	if downloadURL != "" {
		if err := instance.Download(downloadURL); err != nil {
			writeError(w, err)
			return
		}
	}

	resp, err := buildInstanceData(req.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

type patchRequest struct {
	Name       *string  `json:"name"`
	Vendor     *string  `json:"vendor"`
	Version    *string  `json:"version"`
	JavaArgs   []string `json:"java_args"`
	ServerArgs []string `json:"server_args"`
	Memory     *int     `json:"memory"`
}

func patchInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := core.LoadInstanceConfig(id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req patchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Vendor != nil {
		instance.Vendor = *req.Vendor
	}
	if req.Version != nil {
		instance.Version = *req.Version
	}
	if req.JavaArgs != nil {
		instance.JavaArgs = req.JavaArgs
	}
	if req.ServerArgs != nil {
		instance.ServerArgs = req.ServerArgs
	}
	if req.Memory != nil {
		instance.Memory = *req.Memory
	}

	if err := instance.Validate(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if err := core.WriteInstanceConfig(id, instance); err != nil {
		writeError(w, err)
		return
	}
	setCachedInstance(id, instance)

	resp, err := buildInstanceData(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func deleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	evictRCON(id)
	if err := core.DeleteByID(id, sdClient); err != nil {
		writeError(w, err)
		return
	}
	removeCachedInstance(id)
	w.WriteHeader(http.StatusNoContent)
}

func startInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.InstanceConfig) error {
		return inst.Start(sdClient)
	})
}

func stopInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.InstanceConfig) error {
		evictRCON(inst.ID)
		return inst.Stop(sdClient)
	})
}

func restartInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.InstanceConfig) error {
		return inst.Restart(sdClient)
	})
}

func enableInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.InstanceConfig) error {
		return inst.Enable(sdClient)
	})
}

func disableInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.InstanceConfig) error {
		return inst.Disable(sdClient)
	})
}

func instanceAction(w http.ResponseWriter, r *http.Request, fn func(*core.InstanceConfig) error) {
	id := r.PathValue("id")
	instance, err := cachedInstanceOrError(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := fn(instance); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func upgradeInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Vendor  string `json:"vendor"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	v := vendors.Get(req.Vendor)
	if v == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown vendor %q", req.Vendor)})
		return
	}

	instance, err := cachedInstanceOrError(id)
	if err != nil {
		writeError(w, err)
		return
	}

	status, err := sdClient.Status(core.UnitName(id))
	if err != nil {
		writeError(w, err)
		return
	}
	if status.State == "active" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "server must be stopped before upgrading"})
		return
	}

	downloadURL, err := v.DownloadURL(req.Version)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := instance.Download(downloadURL); err != nil {
		writeError(w, err)
		return
	}

	instance.Vendor = req.Vendor
	instance.Version = req.Version
	if err := core.WriteInstanceConfig(id, instance); err != nil {
		writeError(w, err)
		return
	}
	setCachedInstance(id, instance)

	resp, err := buildInstanceData(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func rconInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	instance, err := cachedInstanceOrError(id)
	if err != nil {
		writeError(w, err)
		return
	}
	response, err := sendRCON(instance, req.Command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"response": response})
}
