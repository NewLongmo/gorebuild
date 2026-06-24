package queue

import (
	"reflect"
	"testing"
)

func TestQueueKeys(t *testing.T) {
	tests := []struct {
		name       string
		flashMode  bool
		submitKey  string
		refreshKey string
	}{
		{name: "normal", submitKey: OrderSubmit, refreshKey: OrderRefresh},
		{name: "flash", flashMode: true, submitKey: OrderSubmitFlash, refreshKey: OrderRefreshFlash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubmitKey(tt.flashMode); got != tt.submitKey {
				t.Fatalf("SubmitKey() = %q, want %q", got, tt.submitKey)
			}
			if got := RefreshKey(tt.flashMode); got != tt.refreshKey {
				t.Fatalf("RefreshKey() = %q, want %q", got, tt.refreshKey)
			}
		})
	}
}

func TestPriorityQueueOrder(t *testing.T) {
	if !reflect.DeepEqual(SubmitPriorityKeys(), []string{OrderSubmitFlash, OrderSubmit}) {
		t.Fatalf("SubmitPriorityKeys() = %#v", SubmitPriorityKeys())
	}
	if !reflect.DeepEqual(RefreshPriorityKeys(), []string{OrderRefreshFlash, OrderRefresh}) {
		t.Fatalf("RefreshPriorityKeys() = %#v", RefreshPriorityKeys())
	}
}

func TestAllKeys(t *testing.T) {
	want := []string{OrderSubmit, OrderSubmitFlash, OrderRefresh, OrderRefreshFlash}
	keys := AllKeys()
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("AllKeys() = %#v, want %#v", keys, want)
	}
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			t.Fatalf("AllKeys() contains duplicate key %q in %#v", key, keys)
		}
		seen[key] = true
	}
}
