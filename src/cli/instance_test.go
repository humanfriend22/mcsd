package cli

import (
	"bytes"
	"strings"
	"testing"

	"mcsd/core"
)

func TestWriteInstanceTable(t *testing.T) {
	tests := []struct {
		name      string
		instances []*core.Instance
		// wantLines is the number of newline-terminated lines expected in
		// the output, including the header.
		wantLines int
		// checks are substrings that must appear, in order, one per
		// output line at the same index as they appear in this slice
		// relative to the header (index 0 == header line).
		checkContains []string
		checkNotEmpty bool
	}{
		{
			name:          "empty",
			instances:     []*core.Instance{},
			wantLines:     1,
			checkContains: []string{"ID", "STATUS"},
		},
		{
			name: "healthy active instance",
			instances: []*core.Instance{
				{InstanceConfig: &core.InstanceConfig{ID: "survival"}, State: "active"},
			},
			wantLines:     2,
			checkContains: []string{"survival", "running"},
		},
		{
			name: "healthy inactive instance",
			instances: []*core.Instance{
				{InstanceConfig: &core.InstanceConfig{ID: "creative"}, State: "inactive"},
			},
			wantLines:     2,
			checkContains: []string{"creative", "stopped"},
		},
		{
			name: "degraded instance",
			instances: []*core.Instance{
				{InstanceConfig: &core.InstanceConfig{ID: "broken"}, State: core.InstanceStateError, Error: "bad server-port value"},
			},
			wantLines:     3,
			checkContains: []string{"broken", core.InstanceStateError, "bad server-port value"},
		},
		{
			name: "mixed healthy and degraded, order preserved",
			instances: []*core.Instance{
				{InstanceConfig: &core.InstanceConfig{ID: "first"}, State: "active"},
				{InstanceConfig: &core.InstanceConfig{ID: "second"}, State: core.InstanceStateError, Error: "corrupt config.json"},
				{InstanceConfig: &core.InstanceConfig{ID: "third"}, State: "inactive"},
			},
			wantLines:     5,
			checkContains: []string{"first", "second", "corrupt config.json", "third"},
		},
		{
			name: "degraded instance with empty message writes no trailing blank line",
			instances: []*core.Instance{
				{InstanceConfig: &core.InstanceConfig{ID: "silent"}, State: core.InstanceStateError, Error: ""},
			},
			wantLines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeInstanceTable(&buf, tt.instances); err != nil {
				t.Fatalf("writeInstanceTable() error = %v", err)
			}

			out := buf.String()
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != tt.wantLines {
				t.Fatalf("got %d lines, want %d\noutput:\n%s", len(lines), tt.wantLines, out)
			}

			for _, want := range tt.checkContains {
				if !strings.Contains(out, want) {
					t.Errorf("output missing expected token %q\noutput:\n%s", want, out)
				}
			}
		})
	}
}

func TestWriteInstanceTableDegradedRowShape(t *testing.T) {
	var buf bytes.Buffer
	instances := []*core.Instance{
		{InstanceConfig: &core.InstanceConfig{ID: "broken"}, State: core.InstanceStateError, Error: "invalid server-port \"abc\" in server.properties"},
	}
	if err := writeInstanceTable(&buf, instances); err != nil {
		t.Fatalf("writeInstanceTable() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (header, row, message)\noutput:\n%s", len(lines), buf.String())
	}

	row := lines[1]
	if !strings.Contains(row, "broken") || !strings.Contains(row, core.InstanceStateError) {
		t.Errorf("row %q does not contain id and error state", row)
	}

	msgLine := lines[2]
	if !strings.HasPrefix(msgLine, "  ") {
		t.Errorf("message line %q does not begin with two spaces", msgLine)
	}
	if !strings.Contains(msgLine, "invalid server-port \"abc\" in server.properties") {
		t.Errorf("message line %q missing full error message", msgLine)
	}
}
