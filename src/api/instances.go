package api

import (
	"encoding/json"
	"net/http"

	"mcsd/core"
	"mcsd/vendors"
)

func listInstances(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, getCachedInstances())
}

func getInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

func createInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		core.InstanceConfig
		Ports core.Ports `json:"ports"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	inst, err := core.NewInstance(req.InstanceConfig, req.Ports)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := inst.Create(req.Ports); err != nil {
		writeError(w, err)
		return
	}

	vendor := vendors.Get(inst.Vendor)
	if vendor != nil {
		downloadURL, err := vendor.DownloadURL(inst.Version, inst.Build)
		if err != nil {
			writeError(w, err)
			return
		}
		if downloadURL != "" {
			if err := inst.Download(downloadURL); err != nil {
				writeError(w, err)
				return
			}
		}
	}

	resp, err := core.LoadInstance(inst.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func patchInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	inst, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req core.PatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := inst.Patch(req); err != nil {
		writeError(w, err)
		return
	}

	resp, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func deleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	closeRCON(id)
	if err := core.DeleteInstance(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func startInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.Instance) error {
		return inst.Start()
	})
}

func stopInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.Instance) error {
		closeRCON(inst.ID)
		return inst.Stop()
	})
}

func enableInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.Instance) error {
		return inst.Enable()
	})
}

func disableInstance(w http.ResponseWriter, r *http.Request) {
	instanceAction(w, r, func(inst *core.Instance) error {
		return inst.Disable()
	})
}

func instanceAction(w http.ResponseWriter, r *http.Request, fn func(*core.Instance) error) {
	id := r.PathValue("id")
	inst, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := fn(inst); err != nil {
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
		Build   int    `json:"build"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	v, err := vendors.Require(req.Vendor)
	if err != nil {
		writeError(w, err)
		return
	}

	inst, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := inst.Upgrade(v, req.Version, req.Build); err != nil {
		writeError(w, err)
		return
	}

	resp, err := core.LoadInstance(id)
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

	inst, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}
	response, err := sendRCON(inst, req.Command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"response": response})
}
