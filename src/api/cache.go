package api

import (
	"sync"
	"time"

	"mcsd/core"
)

// This is very crude basic cache system just in case multiple clients are polling.

const instanceCacheInterval = 2 * time.Second

var (
	instancesMu     sync.RWMutex
	cachedInstances []core.InstanceResult
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

func getCachedInstances() []core.InstanceResult {
	instancesMu.RLock()
	defer instancesMu.RUnlock()
	return cachedInstances
}

// closeStaleRCON closes any pooled RCON connection whose instance no longer
// exists, e.g. deleted via the CLI, which the pool has no other way to learn
// about. It reads only .ID off each result, so a degraded entry (nil
// Instance) is handled the same as a healthy one.
func closeStaleRCON(live []core.InstanceResult) {
	liveIDs := make(map[string]struct{}, len(live))
	for _, res := range live {
		liveIDs[res.ID] = struct{}{}
	}
	rconConnections.Range(func(key, _ any) bool {
		id := key.(string)
		if _, ok := liveIDs[id]; !ok {
			closeRCON(id)
		}
		return true
	})
}
