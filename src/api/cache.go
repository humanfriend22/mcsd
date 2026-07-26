package api

import (
	"sync"
	"time"

	"mcsd/core"
)

const instanceCacheInterval = 2 * time.Second

var (
	instancesMu     sync.RWMutex
	cachedInstances []*core.Instance
)

// startInstanceCache refreshes the instance list on a timer so concurrent
// clients share one read instead of each poll re-deriving state via systemd/D-Bus.
func startInstanceCache() {
	refreshInstances()
	for range time.Tick(instanceCacheInterval) {
		refreshInstances()
	}
}

func refreshInstances() {
	results, err := core.ListInstances()
	if err != nil {
		return // keep last good cache on transient error
	}

	instancesMu.Lock()
	cachedInstances = results
	instancesMu.Unlock()

	closeStaleRCON(results)
}

func getCachedInstances() []*core.Instance {
	instancesMu.RLock()
	defer instancesMu.RUnlock()
	return cachedInstances
}

// closeStaleRCON closes any pooled RCON connection whose instance no longer
// exists, e.g. deleted via the CLI, which the pool has no other way to learn about.
func closeStaleRCON(live []*core.Instance) {
	liveIDs := make(map[string]struct{}, len(live))
	for _, inst := range live {
		liveIDs[inst.ID] = struct{}{}
	}
	rconConnections.Range(func(key, _ any) bool {
		id := key.(string)
		if _, ok := liveIDs[id]; !ok {
			closeRCON(id)
		}
		return true
	})
}
