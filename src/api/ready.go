package api

import (
	"net/http"

	"mcsd/core"
)

type readyResponse struct {
	Ready bool   `json:"ready"`
	Error string `json:"error"`
}

func checkReady(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := core.LoadInstance(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := inst.EnsureStartReady(); err != nil {
		writeJSON(w, http.StatusOK, readyResponse{Ready: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, readyResponse{Ready: true, Error: ""})
}
