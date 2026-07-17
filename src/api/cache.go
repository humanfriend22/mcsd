package api

import (
	"fmt"
	"sync"
	"time"

	"mcsd/core"
)

var (
	cacheMu          sync.RWMutex
	cachedInstances  map[string]*core.InstanceConfig
	cachedGlobal     *core.Config
	lastRecache      time.Time
	recacheMinInterval = time.Second
)

func forceRecache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	doRecacheLocked()
	lastRecache = time.Now()
}

func recache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if time.Since(lastRecache) < recacheMinInterval {
		return
	}
	doRecacheLocked()
	lastRecache = time.Now()
}

func doRecacheLocked() {
	cfg, err := core.LoadConfig()
	if err == nil {
		cachedGlobal = cfg
	}

	ids, err := core.ListInstanceConfigs()
	if err != nil {
		return
	}
	instances := make(map[string]*core.InstanceConfig, len(ids))
	for _, id := range ids {
		inst, err := core.LoadInstanceConfig(id)
		if err != nil {
			continue
		}
		instances[id] = inst
	}
	cachedInstances = instances
}

func getCachedIDs() []string {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	ids := make([]string, 0, len(cachedInstances))
	for id := range cachedInstances {
		ids = append(ids, id)
	}
	return ids
}

func getCachedInstance(id string) (*core.InstanceConfig, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	inst, ok := cachedInstances[id]
	return inst, ok
}

func getCachedInstances() map[string]*core.InstanceConfig {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return cachedInstances
}

func getCachedGlobalConfig() *core.Config {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return cachedGlobal
}

func setCachedInstance(id string, inst *core.InstanceConfig) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cachedInstances == nil {
		cachedInstances = make(map[string]*core.InstanceConfig)
	}
	cachedInstances[id] = inst
}

func removeCachedInstance(id string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	delete(cachedInstances, id)
}

func cachedInstanceOrError(id string) (*core.InstanceConfig, error) {
	inst, ok := getCachedInstance(id)
	if !ok {
		return nil, fmt.Errorf("instance %q not found", id)
	}
	return inst, nil
}
