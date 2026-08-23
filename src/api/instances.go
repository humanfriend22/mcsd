package api

import (
	"encoding/json"
	"net/http"

	"mcsd/core"
	"mcsd/vendors"
)

// degradedInstance is the API layer's own error representation for an
// instance id whose config or ports could not be loaded — core no longer
// fabricates one. It embeds *core.Instance anonymously so the JSON field set
// is unchanged from a healthy instance (flattened by anonymous embedding),
// plus an Error field. web/app/services/api.ts reads instance.error off this
// same shape, so the wire contract stays byte-identical to before.
type degradedInstance struct {
	*core.Instance
	Error string `json:"error,omitempty"`
}

// newDegradedInstance builds the API's degraded payload for an id that
// failed to load. No config value is fabricated beyond ID/Name (both set to
// id, since no real name is available) and the shared degraded state.
func newDegradedInstance(id string, err error) degradedInstance {
	return degradedInstance{
		Instance: &core.Instance{
			InstanceConfig: &core.InstanceConfig{ID: id, Name: id},
			State:          core.InstanceStateError,
		},
		Error: err.Error(),
	}
}

func listInstances(w http.ResponseWriter, r *http.Request) {
	results := getCachedInstances()
	payload := make([]any, 0, len(results))
	for _, res := range results {
		if res.Err != nil {
			payload = append(payload, newDegradedInstance(res.ID, res.Err))
			continue
		}
		payload = append(payload, res.Instance)
	}
	writeJSON(w, http.StatusOK, payload)
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

	inst := &core.Instance{InstanceConfig: &req.InstanceConfig, Ports: req.Ports}
	if err := inst.Validate(); err != nil {
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
