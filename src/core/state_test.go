package core

import (
	"encoding/json"
	"testing"
	"time"
)

// TestInstanceStateMarshalsFlattened locks the JSON wire contract
// web/app/services/api.ts depends on: an Instance's embedded InstanceState
// must marshal with state/enabled/active_since/memory_used as top-level
// keys, not nested under an "InstanceState" key. This is what a future
// author adding a json tag or a field name to the Instance embed would
// silently break — the frontend reads these keys flat and would start
// getting undefined for all four fields with no compile-time signal on
// either side of the wire.
func TestInstanceStateMarshalsFlattened(t *testing.T) {
	since := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	instance := Instance{
		InstanceConfig: &InstanceConfig{ID: "state-wire", Name: "state-wire"},
		InstanceState: InstanceState{
			State:       "active",
			Enabled:     true,
			ActiveSince: &since,
			MemoryUsed:  512,
		},
	}

	raw, err := json.Marshal(instance)
	if err != nil {
		t.Fatalf("json.Marshal(instance) error = %v, want nil", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(raw) error = %v, want nil", err)
	}

	for _, key := range []string{"state", "enabled", "active_since", "memory_used"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("decoded payload missing top-level key %q — web/app/services/api.ts reads Instance.%s at the top level, not nested", key, key)
		}
	}

	if _, ok := decoded["InstanceState"]; ok {
		t.Errorf("decoded payload has a nested %q key — the embed must stay anonymous and untagged so encoding/json flattens it; a named or tagged embed would silently move these fields out from under web/app/services/api.ts", "InstanceState")
	}
}

// TestInstanceStateActiveSinceNilMarshalsNull asserts that a nil ActiveSince
// marshals to JSON null (matching the `string | null` type declared for
// active_since in web/app/services/api.ts), rather than being omitted or
// rendered as a zero-value timestamp string.
func TestInstanceStateActiveSinceNilMarshalsNull(t *testing.T) {
	instance := Instance{
		InstanceConfig: &InstanceConfig{ID: "state-null", Name: "state-null"},
		InstanceState:  InstanceState{State: "inactive"},
	}

	raw, err := json.Marshal(instance)
	if err != nil {
		t.Fatalf("json.Marshal(instance) error = %v, want nil", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(raw) error = %v, want nil", err)
	}

	value, ok := decoded["active_since"]
	if !ok {
		t.Fatalf("decoded payload missing \"active_since\" key, want present with a null value")
	}
	if value != nil {
		t.Errorf("active_since = %#v, want JSON null for a nil ActiveSince", value)
	}
}

// TestLoadInstanceStateNilSDManagerReturnsInactiveDefault asserts that
// loadInstanceState does not panic when SDManager is nil (a developer
// machine with no systemd, the same environment
// TestListInstancesEmptyIsNotFailure already relies on) and instead returns
// the inactive default: not enabled, no active-since timestamp, zero memory.
func TestLoadInstanceStateNilSDManagerReturnsInactiveDefault(t *testing.T) {
	original := SDManager
	SDManager = nil
	defer func() { SDManager = original }()

	got := loadInstanceState("state-nil-sdmanager")

	want := InstanceState{State: "inactive"}
	if got != want {
		t.Errorf("loadInstanceState() with nil SDManager = %+v, want %+v", got, want)
	}
}
