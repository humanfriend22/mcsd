package cli

import (
	"bytes"
	"errors"
	"testing"

	"mcsd/core"
)

// TestWriteInstanceTableMixed drives writeInstanceTable with a hand-built
// mixed slice — one healthy success entry, one failure entry carrying a real
// error — and asserts on the full rendered output. Because the writer is a
// tabwriter, column padding is computed from content, so this compares
// against the exact expected string rather than substring-matching.
func TestWriteInstanceTableMixed(t *testing.T) {
	results := []core.InstanceResult{
		{
			ID: "healthy-one",
			Instance: &core.Instance{
				InstanceConfig: &core.InstanceConfig{ID: "healthy-one", Name: "healthy-one"},
				InstanceState:  core.InstanceState{State: "active"},
			},
		},
		{
			ID:  "broken-two",
			Err: errors.New("config.json: corrupt"),
		},
	}

	var buf bytes.Buffer
	if err := writeInstanceTable(&buf, results); err != nil {
		t.Fatalf("writeInstanceTable() error = %v, want nil", err)
	}

	want := "ID                      STATUS\n" +
		"healthy-one             running\n" +
		"broken-two              error\n" +
		"  config.json: corrupt  \n"

	if got := buf.String(); got != want {
		t.Errorf("writeInstanceTable() output mismatch:\n got:  %q\n want: %q", got, want)
	}
}

// TestWriteInstanceTableEmptyMessage asserts that a failure entry whose
// error renders as an empty string produces the status row and no trailing
// indented line.
func TestWriteInstanceTableEmptyMessage(t *testing.T) {
	results := []core.InstanceResult{
		{
			ID:  "broken-empty",
			Err: errors.New(""),
		},
	}

	var buf bytes.Buffer
	if err := writeInstanceTable(&buf, results); err != nil {
		t.Fatalf("writeInstanceTable() error = %v, want nil", err)
	}

	want := "ID            STATUS\n" +
		"broken-empty  error\n"

	if got := buf.String(); got != want {
		t.Errorf("writeInstanceTable() output mismatch:\n got:  %q\n want: %q", got, want)
	}
}
