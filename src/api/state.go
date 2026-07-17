package api

import (
	"net/http"
	"time"

	"mcsd/core"
)

type instanceRunState struct {
	State       string     `json:"state"`
	Enabled     bool       `json:"enabled"`
	ActiveSince *time.Time `json:"active_since,omitempty"`
	MemoryUsed  int        `json:"memory_used,omitempty"`
}

func getAllStates(w http.ResponseWriter, r *http.Request) {
	ids := getCachedIDs()
	result := make(map[string]instanceRunState, len(ids))
	for _, id := range ids {
		unit := core.UnitName(id)
		rs := instanceRunState{
			State:   "inactive",
			Enabled: sdClient.IsEnabled(unit),
		}
		if status, err := sdClient.Status(unit); err == nil {
			rs.State = status.State
			if status.State == "active" {
				if since, err := sdClient.ActiveSince(unit); err == nil {
					rs.ActiveSince = &since
				}
				if bytes, err := sdClient.MemoryUsedBytes(unit); err == nil && bytes > 0 {
					rs.MemoryUsed = int(bytes / (1024 * 1024))
				}
			}
		}
		result[id] = rs
	}
	writeJSON(w, http.StatusOK, result)
}
