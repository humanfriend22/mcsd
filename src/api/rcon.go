package api

import (
	"fmt"
	"sync"
	"time"

	"mcsd/core"
)

const rconIdleTTL = 30 * time.Second

type rcon struct {
	mu     sync.Mutex
	client *core.RCONClient
	timer  *time.Timer
}

var rconConnections sync.Map // string → *rcon

func sendRCON(instance *core.Instance, command string) (string, error) {
	val, _ := rconConnections.LoadOrStore(instance.ID, &rcon{})
	entry := val.(*rcon)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Connect to instance if not already connected
	if entry.client == nil {
		client, err := core.DialRCON(fmt.Sprintf("localhost:%d", instance.Ports.RCON), instance.Ports.RCONPassword)
		if err != nil {
			return "", err
		}
		entry.client = client
	}

	// Close connection if idle
	if entry.timer != nil {
		entry.timer.Reset(rconIdleTTL)
	} else {
		entry.timer = time.AfterFunc(rconIdleTTL, func() {
			entry.mu.Lock()
			defer entry.mu.Unlock()
			if entry.client != nil {
				entry.client.Close()
				entry.client = nil
			}
		})
	}

	resp, err := entry.client.Send(command)
	if err != nil {
		entry.client.Close()
		entry.client = nil
		return "", err
	}
	return resp, nil
}

func closeRCON(id string) {
	if val, ok := rconConnections.LoadAndDelete(id); ok {
		entry := val.(*rcon)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if entry.timer != nil {
			entry.timer.Stop()
		}
		if entry.client != nil {
			entry.client.Close()
			entry.client = nil
		}
	}
}

// closeStaleRCON closes any pooled RCON connection whose instance no longer
// exists, e.g. deleted via the CLI, which the pool has no other way to learn
// about. It reads only .ID off each result, so a degraded entry (nil
// Instance) is handled the same as a healthy one.
func closeStaleRCONs(irs []core.InstanceResult) {
	ids := make(map[string]struct{}, len(irs))
	for _, ir := range irs {
		ids[ir.ID] = struct{}{}
	}
	rconConnections.Range(func(key, _ any) bool {
		id := key.(string)
		if _, ok := ids[id]; !ok {
			closeRCON(id)
		}
		return true
	})
}
