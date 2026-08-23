package api

import (
	"encoding/json"
	"errors"
	"testing"
)

// TestNewDegradedInstanceMarshaling marshals the degraded DTO built by
// newDegradedInstance and asserts it carries every key
// web/app/components/ServerCard.vue reads on a degraded card — at minimum
// id, name, vendor, version, memory, ports, enabled, memory_used,
// active_since — plus the expected state/error/name values. This is the
// guard that keeps web/app/services/api.ts untouched: a future change to
// this DTO's shape should be caught here before it reaches the frontend.
func TestNewDegradedInstanceMarshaling(t *testing.T) {
	dto := newDegradedInstance("broken-instance", errors.New("config.json: corrupt"))

	data, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("json.Marshal(dto) error = %v, want nil", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(data) error = %v, want nil", err)
	}

	wantKeys := []string{
		"id", "name", "vendor", "version", "memory",
		"ports", "enabled", "memory_used", "active_since",
	}
	for _, key := range wantKeys {
		if _, ok := got[key]; !ok {
			t.Errorf("degraded payload missing key %q (present: %v) — web/app/services/api.ts expects it on every instance", key, got)
		}
	}

	if state, _ := got["state"].(string); state != "error" {
		t.Errorf(`degraded payload "state" = %q, want "error"`, state)
	}
	if errMsg, _ := got["error"].(string); errMsg != "config.json: corrupt" {
		t.Errorf(`degraded payload "error" = %q, want "config.json: corrupt"`, errMsg)
	}
	if name, _ := got["name"].(string); name != "broken-instance" {
		t.Errorf(`degraded payload "name" = %q, want "broken-instance"`, name)
	}
	if id, _ := got["id"].(string); id != "broken-instance" {
		t.Errorf(`degraded payload "id" = %q, want "broken-instance"`, id)
	}
	if _, ok := got["ports"].(map[string]any); !ok {
		t.Errorf(`degraded payload "ports" = %v, want a JSON object`, got["ports"])
	}
}
