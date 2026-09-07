package api

import (
	"sync"
	"sync/atomic"
	"time"

	"mcsd/core"
)

// Debounced pull-through cache: every request calls getCachedInstances.
// Whichever request finds the cache stale is the one that refreshes it (via
// CompareAndSwap on refreshing); everyone else just reads whatever is
// currently cached, so concurrent pollers never pile up duplicate
// systemd/D-Bus lookups on top of each other.

const instanceCacheInterval = 1 * time.Second

var (
	cachedInstancesMutex     sync.RWMutex
	cachedInstances []core.InstanceResult

	lastRefresh     time.Time
	refreshing atomic.Bool
)

func getCachedInstances() []core.InstanceResult {
	cachedInstancesMutex.RLock()
	cold := lastRefresh.IsZero()
	stale := time.Since(lastRefresh) >= instanceCacheInterval
	cachedInstancesMutex.RUnlock()

	if cold {
		// Nothing cached yet: block so the caller doesn't see a nil list.
		// If another request already grabbed the refresh, briefly wait for
		// it instead of racing back an empty slice.
		if refreshing.CompareAndSwap(false, true) {
			refreshInstances()
			refreshing.Store(false)
		} else {
			for refreshing.Load() {
				time.Sleep(time.Millisecond)
			}
		}
	} else if stale && refreshing.CompareAndSwap(false, true) {
		go func() {
			defer refreshing.Store(false)
			refreshInstances()
		}()
	}

	cachedInstancesMutex.RLock()
	defer cachedInstancesMutex.RUnlock()
	return cachedInstances
}

func refreshInstances() {
	instances, err := core.ListInstances()
	if err != nil {
		return // keep last good cache on transient error
	}

	cachedInstancesMutex.Lock()
	cachedInstances = instances
	lastRefresh = time.Now()
	cachedInstancesMutex.Unlock()

	closeStaleRCONs(instances)
}
