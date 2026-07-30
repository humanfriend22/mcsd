package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "mcsd/utils"
)

// TestTracerDegradedInstancePath walks the whole degraded-instance path in one
// function: a corrupt server-port value in server.properties becomes a typed
// *ValidationError from ReadPorts, that error becomes a degraded list entry
// via buildInstanceList, and the entry serializes to the JSON contract the
// API will emit.
func TestTracerDegradedInstancePath(t *testing.T) {
	dir := t.TempDir()
	contents := "server-port=not-a-number\n" +
		"rcon.port=25575\n" +
		"rcon.password=S3ntinel-Pass-Do-Not-Leak\n"
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(contents), 0640); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ReadPorts(dir)
	if err == nil {
		t.Fatalf("expected ReadPorts to return an error for a corrupt server-port value")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(valErr.Error(), "server-port") {
		t.Fatalf("expected error message to name server-port, got: %s", valErr.Error())
	}
	if strings.Contains(valErr.Error(), "S3ntinel-Pass-Do-Not-Leak") {
		t.Fatalf("error message leaked the rcon.password sentinel: %s", valErr.Error())
	}

	ids := []string{"first", "second", "third"}
	loader := func(id string) (*Instance, error) {
		if id == "second" {
			return nil, err
		}
		return &Instance{InstanceConfig: &InstanceConfig{ID: id, Name: id}, State: "inactive"}, nil
	}

	instances := buildInstanceList(ids, loader)
	if len(instances) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(instances))
	}
	if instances[0].ID != "first" || instances[0].State != "inactive" {
		t.Fatalf("expected first entry unchanged, got %+v", instances[0])
	}
	if instances[2].ID != "third" || instances[2].State != "inactive" {
		t.Fatalf("expected third entry unchanged, got %+v", instances[2])
	}
	degraded := instances[1]
	if degraded.ID != "second" {
		t.Fatalf("expected degraded entry id 'second', got %q", degraded.ID)
	}
	if degraded.State != InstanceStateError {
		t.Fatalf("expected degraded entry State %q, got %q", InstanceStateError, degraded.State)
	}
	if degraded.Error == "" {
		t.Fatalf("expected degraded entry to carry a non-empty Error message")
	}

	encoded, marshalErr := json.Marshal(degraded)
	if marshalErr != nil {
		t.Fatalf("json.Marshal degraded entry: %v", marshalErr)
	}
	body := string(encoded)
	if !strings.Contains(body, `"state":"error"`) {
		t.Fatalf("expected marshaled JSON to contain state:error, got: %s", body)
	}
	if !strings.Contains(body, `"error":`) {
		t.Fatalf("expected marshaled JSON to contain an error member, got: %s", body)
	}
	if !strings.Contains(body, `"id":"second"`) {
		t.Fatalf("expected marshaled JSON to contain id:second, got: %s", body)
	}
}
