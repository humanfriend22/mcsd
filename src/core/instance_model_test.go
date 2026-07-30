package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

// TestBuildInstanceList pins the count, order, and duplicate-id invariants
// (D-01): the returned length always equals len(ids), input order is
// preserved position-for-position, and duplicate ids never merge.
func TestBuildInstanceList(t *testing.T) {
	healthy := func(id string) (*Instance, error) {
		return &Instance{InstanceConfig: &InstanceConfig{ID: id, Name: id}, State: "inactive"}, nil
	}
	failing := func(id string) (*Instance, error) {
		return nil, &ServerError{Message: fmt.Sprintf("boom loading %s", id)}
	}

	t.Run("zero ids", func(t *testing.T) {
		instances := buildInstanceList(nil, healthy)
		if instances == nil {
			t.Fatalf("expected a non-nil empty slice, got nil")
		}
		if len(instances) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(instances))
		}
	})

	t.Run("one failing id", func(t *testing.T) {
		instances := buildInstanceList([]string{"only"}, failing)
		if len(instances) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(instances))
		}
		if instances[0].State != InstanceStateError || instances[0].ID != "only" {
			t.Fatalf("expected degraded entry for 'only', got %+v", instances[0])
		}
	})

	t.Run("all ids fail", func(t *testing.T) {
		ids := []string{"a", "b", "c"}
		instances := buildInstanceList(ids, failing)
		if len(instances) != len(ids) {
			t.Fatalf("expected %d entries, got %d", len(ids), len(instances))
		}
		for i, id := range ids {
			if instances[i].ID != id {
				t.Fatalf("position %d: expected id %q, got %q", i, id, instances[i].ID)
			}
			if instances[i].State != InstanceStateError {
				t.Fatalf("position %d: expected degraded state, got %q", i, instances[i].State)
			}
		}
	})

	t.Run("interleaved healthy and degraded", func(t *testing.T) {
		ids := []string{"h1", "f1", "h2", "f2", "h3"}
		load := func(id string) (*Instance, error) {
			if strings.HasPrefix(id, "f") {
				return failing(id)
			}
			return healthy(id)
		}
		instances := buildInstanceList(ids, load)
		if len(instances) != len(ids) {
			t.Fatalf("expected %d entries, got %d", len(ids), len(instances))
		}
		for i, id := range ids {
			if instances[i].ID != id {
				t.Fatalf("position %d: expected id %q, got %q (order not preserved)", i, id, instances[i].ID)
			}
			wantDegraded := strings.HasPrefix(id, "f")
			gotDegraded := instances[i].State == InstanceStateError
			if wantDegraded != gotDegraded {
				t.Fatalf("position %d (id %q): expected degraded=%v, got degraded=%v", i, id, wantDegraded, gotDegraded)
			}
		}
	})

	t.Run("duplicate ids never merge", func(t *testing.T) {
		ids := []string{"dup", "dup", "dup"}
		instances := buildInstanceList(ids, failing)
		if len(instances) != 3 {
			t.Fatalf("expected 3 separate entries for duplicate ids, got %d", len(instances))
		}
		for i, inst := range instances {
			if inst.ID != "dup" {
				t.Fatalf("position %d: expected id 'dup', got %q", i, inst.ID)
			}
		}
	})
}

// TestLogInstanceLoadFailureDedupe pins the log-rate invariant (T-01-03):
// an unchanged repeated failure logs once, a changed message logs again, and
// clearInstanceLoadFailure re-arms logging for a subsequent recurrence.
func TestLogInstanceLoadFailureDedupe(t *testing.T) {
	var buf bytes.Buffer
	prevOutput := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOutput)
		log.SetFlags(prevFlags)
	})

	const id = "dedupe-test-instance"
	t.Cleanup(func() { clearInstanceLoadFailure(id) })
	clearInstanceLoadFailure(id) // isolate from any prior subtest state

	countLines := func() int {
		s := strings.TrimRight(buf.String(), "\n")
		if s == "" {
			return 0
		}
		return len(strings.Split(s, "\n"))
	}

	logInstanceLoadFailure(id, &ServerError{Message: "disk error A"})
	logInstanceLoadFailure(id, &ServerError{Message: "disk error A"})
	if got := countLines(); got != 1 {
		t.Fatalf("expected 1 log line for two identical consecutive failures, got %d: %q", got, buf.String())
	}

	logInstanceLoadFailure(id, &ServerError{Message: "disk error B"})
	if got := countLines(); got != 2 {
		t.Fatalf("expected 2 log lines once the message changes, got %d: %q", got, buf.String())
	}

	clearInstanceLoadFailure(id)
	logInstanceLoadFailure(id, &ServerError{Message: "disk error B"})
	if got := countLines(); got != 3 {
		t.Fatalf("expected logging to re-arm after clearInstanceLoadFailure, got %d lines: %q", got, buf.String())
	}
}
