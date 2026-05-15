package direct

import (
	"reflect"
	"testing"
)

func TestComputeRemovedTagIDs(t *testing.T) {
	current := []v4TagObj{
		{Tag: "Лидген", TagID: 1},
		{Tag: "Вторичка", TagID: 2},
		{Tag: "topic:vtorichka", TagID: 3},
		{Tag: "channel:search", TagID: 4},
	}
	removeSet := map[string]struct{}{
		"topic:vtorichka": {},
		"channel:search":  {},
	}
	newIDs, removed := computeRemovedTagIDs(current, removeSet)

	wantIDs := []int64{1, 2}
	if !reflect.DeepEqual(newIDs, wantIDs) {
		t.Errorf("newIDs = %v, want %v", newIDs, wantIDs)
	}
	wantRemoved := []string{"topic:vtorichka", "channel:search"}
	if !reflect.DeepEqual(removed, wantRemoved) {
		t.Errorf("removed = %v, want %v", removed, wantRemoved)
	}
}

func TestComputeRemovedTagIDs_CaseInsensitive(t *testing.T) {
	current := []v4TagObj{
		{Tag: "Topic:Vtorichka", TagID: 1},
		{Tag: "Лидген", TagID: 2},
	}
	removeSet := map[string]struct{}{"topic:vtorichka": {}}

	newIDs, removed := computeRemovedTagIDs(current, removeSet)
	if !reflect.DeepEqual(newIDs, []int64{2}) {
		t.Errorf("newIDs = %v, want [2]", newIDs)
	}
	if !reflect.DeepEqual(removed, []string{"Topic:Vtorichka"}) {
		t.Errorf("removed = %v, want [Topic:Vtorichka]", removed)
	}
}

func TestComputeRemovedTagIDs_NoMatch(t *testing.T) {
	current := []v4TagObj{
		{Tag: "Лидген", TagID: 1},
		{Tag: "Вторичка", TagID: 2},
	}
	removeSet := map[string]struct{}{"not-present": {}}

	newIDs, removed := computeRemovedTagIDs(current, removeSet)
	wantIDs := []int64{1, 2}
	if !reflect.DeepEqual(newIDs, wantIDs) {
		t.Errorf("newIDs = %v, want %v", newIDs, wantIDs)
	}
	if removed != nil {
		t.Errorf("removed = %v, want nil", removed)
	}
}

func TestComputeRemovedTagIDs_RemoveAll(t *testing.T) {
	current := []v4TagObj{
		{Tag: "a", TagID: 1},
		{Tag: "b", TagID: 2},
	}
	removeSet := map[string]struct{}{"a": {}, "b": {}}

	newIDs, removed := computeRemovedTagIDs(current, removeSet)
	if newIDs != nil {
		t.Errorf("newIDs = %v, want nil (caller normalizes to []int64{})", newIDs)
	}
	if !reflect.DeepEqual(removed, []string{"a", "b"}) {
		t.Errorf("removed = %v, want [a, b]", removed)
	}
}

func TestComputeRemovedTagIDs_Empty(t *testing.T) {
	newIDs, removed := computeRemovedTagIDs(nil, map[string]struct{}{"x": {}})
	if newIDs != nil || removed != nil {
		t.Errorf("expected (nil, nil), got (%v, %v)", newIDs, removed)
	}
}
