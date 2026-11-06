package realtime

import (
	"context"
	"testing"
)

func TestHubFromContextPrefersInjectedHub(t *testing.T) {
	ResetHub()
	defer ResetHub()
	injected := NewHub()
	ctx := WithHub(context.Background(), injected)
	if got := HubFromContext(ctx); got != injected {
		t.Fatalf("HubFromContext must return injected hub, got %p want %p", got, injected)
	}
	if got := HubFromContext(context.Background()); got != DefaultHub() {
		t.Fatalf("HubFromContext without injection must fall back to DefaultHub")
	}
	if got := HubFromContext(WithHub(context.Background(), nil)); got != DefaultHub() {
		t.Fatalf("nil hub must be ignored (fallback to DefaultHub)")
	}
}
