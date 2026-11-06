package observability

import (
	"context"
	"testing"
)

func TestLoggerFromContextPrefersInjectedLogger(t *testing.T) {
	logger, err := NewLogger("test", "error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	ctx := WithLogger(context.Background(), logger)
	if got := FromContext(ctx); got != logger {
		t.Fatalf("FromContext must return injected logger")
	}
	if got := FromContext(context.Background()); got != L() {
		t.Fatalf("FromContext without injection must fall back to L()")
	}
	if got := FromContext(WithLogger(context.Background(), nil)); got != L() {
		t.Fatalf("nil logger must be ignored")
	}
}
