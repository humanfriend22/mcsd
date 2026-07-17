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

var rconCache sync.Map // string → *rconEntry

func sendRCON(instance *core.InstanceConfig, command string) (string, error) {
	val, _ := rconCache.LoadOrStore(instance.ID, &rconEntry{})
	entry := val.(*rconEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.client == nil {
		ports, err := instance.ReadPorts()
		if err != nil {
			return "", fmt.Errorf("read ports: %w", err)
		}
		client, err := core.DialRCON(fmt.Sprintf("localhost:%d", ports.RCON), ports.RCONPassword)
		if err != nil {
			return "", err
		}
		entry.client = client
	}

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

func evictRCON(id string) {
	if val, ok := rconCache.LoadAndDelete(id); ok {
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
