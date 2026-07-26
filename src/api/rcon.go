package api

import (
	"fmt"
	"sync"
	"time"

	"mcsd/core"
)

const rconIdleTTL = 30 * time.Second

type rconEntry struct {
	mu     sync.Mutex
	client *core.RCONClient
	timer  *time.Timer
}

var rconConnections sync.Map // string → *rconEntry

func sendRCON(instance *core.Instance, command string) (string, error) {
	val, _ := rconConnections.LoadOrStore(instance.ID, &rconEntry{})
	entry := val.(*rconEntry)

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
		entry := val.(*rconEntry)
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
