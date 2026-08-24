package core

import (
	"errors"
	"testing"

	. "mcsd/utils"
)

// TestLoadInstanceConfigRejectsMalformedID asserts that LoadInstanceConfig
// rejects a malformed or traversal-shaped id with a *ValidationError before
// ever touching the filesystem, while still letting a well-formed but
// absent id fall through to the *NotFoundError filesystem lookup (the
// over-rejection guard).
func TestLoadInstanceConfigRejectsMalformedID(t *testing.T) {
	cases := []struct {
		name         string
		id           string
		wantNotFound bool
	}{
		{name: "traversal", id: "../../../etc/passwd"},
		{name: "slash", id: "foo/bar"},
		{name: "empty", id: ""},
		{name: "dot", id: ".."},
		{name: "well-formed but absent", id: "qt05o-absent-instance", wantNotFound: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadInstanceConfig(tc.id)

			if cfg != nil {
				t.Errorf("LoadInstanceConfig(%q) config = %v, want nil", tc.id, cfg)
			}
			if err == nil {
				t.Fatalf("LoadInstanceConfig(%q) err = nil, want non-nil", tc.id)
			}

			if tc.wantNotFound {
				var notFound *NotFoundError
				if !errors.As(err, &notFound) {
					t.Errorf("LoadInstanceConfig(%q) err = %T, want *NotFoundError", tc.id, err)
				}
				return
			}

			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Errorf("LoadInstanceConfig(%q) err = %T, want *ValidationError", tc.id, err)
			}
		})
	}
}
