package core

import (
	"errors"
	"testing"
)

// TestBuildInstanceListAllSuccess asserts that a loader which succeeds for
// every id yields one success entry per id, in input order, with a non-nil
// Instance and a nil Err.
func TestBuildInstanceListAllSuccess(t *testing.T) {
	ids := []string{"tbl-alpha", "tbl-bravo", "tbl-charlie"}
	load := func(id string) (*Instance, error) {
		return &Instance{InstanceConfig: &InstanceConfig{ID: id, Name: id}}, nil
	}

	results := buildInstanceList(ids, load)

	if len(results) != len(ids) {
		t.Fatalf("got %d results, want %d", len(results), len(ids))
	}
	for i, id := range ids {
		res := results[i]
		if res.ID != id {
			t.Errorf("result[%d].ID = %q, want %q (order not preserved)", i, res.ID, id)
		}
		if res.Instance == nil {
			t.Errorf("result[%d].Instance = nil, want non-nil for success entry %q", i, id)
		}
		if res.Err != nil {
			t.Errorf("result[%d].Err = %v, want nil for success entry %q", i, res.Err, id)
		}
	}
}

// TestBuildInstanceListMixedFailure asserts that a loader which fails for
// exactly one id yields a failure entry for that id only — nil Instance, Err
// equal to the loader's error — while its neighbours stay successes. It also
// asserts the returned slice length always equals the input id count.
func TestBuildInstanceListMixedFailure(t *testing.T) {
	ids := []string{"tbl-mix-one", "tbl-mix-two", "tbl-mix-three"}
	loadErr := errors.New("config.json: corrupt")
	load := func(id string) (*Instance, error) {
		if id == "tbl-mix-two" {
			return nil, loadErr
		}
		return &Instance{InstanceConfig: &InstanceConfig{ID: id, Name: id}}, nil
	}

	results := buildInstanceList(ids, load)

	if len(results) != len(ids) {
		t.Fatalf("got %d results, want %d (a failure must neither drop nor duplicate an entry)", len(results), len(ids))
	}

	for i, id := range ids {
		res := results[i]
		if res.ID != id {
			t.Errorf("result[%d].ID = %q, want %q", i, res.ID, id)
		}
		if id == "tbl-mix-two" {
			if res.Instance != nil {
				t.Errorf("result[%d] (%q) Instance = %+v, want nil for failure entry — no stand-in object should be manufactured", i, id, res.Instance)
			}
			if !errors.Is(res.Err, loadErr) {
				t.Errorf("result[%d] (%q) Err = %v, want the exact loader error %v", i, id, res.Err, loadErr)
			}
			continue
		}
		if res.Instance == nil {
			t.Errorf("result[%d] (%q) Instance = nil, want non-nil for neighbouring success entry", i, id)
		}
		if res.Err != nil {
			t.Errorf("result[%d] (%q) Err = %v, want nil for neighbouring success entry", i, id, res.Err)
		}
	}
}

// TestListInstancesEmptyIsNotFailure asserts the signature and the
// empty-vs-failure distinction, not per-instance loading: on a developer
// machine the instances directory does not exist, so ListInstanceConfigs
// returns an empty list with a nil error, and ListInstances must return a
// non-nil empty slice and a nil error — distinguishing "no servers found"
// from a whole-list failure (unreadable instances dir).
func TestListInstancesEmptyIsNotFailure(t *testing.T) {
	results, err := ListInstances()
	if err != nil {
		t.Fatalf("ListInstances() error = %v, want nil", err)
	}
	if results == nil {
		t.Fatalf("ListInstances() returned a nil slice, want a non-nil empty slice")
	}
}
