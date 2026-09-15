package buildinfo

import "testing"

func TestSummaryDevBuild(t *testing.T) {
	Version, Commit, BuildTime = "dev", "", ""
	defer func() { Version, Commit, BuildTime = "dev", "", "" }()
	if got := Summary(); got != "dev" {
		t.Fatalf("expected dev, got %q", got)
	}
	if Committed() {
		t.Fatal("dev build should not be Committed")
	}
}

func TestSummaryInjectedBuild(t *testing.T) {
	Version, Commit, BuildTime = "v1.0.0-2-g1a2b3c4", "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", "2026-09-16T00:00:00Z"
	defer func() { Version, Commit, BuildTime = "dev", "", "" }()
	got := Summary()
	if got != "v1.0.0-2-g1a2b3c4 (commit 1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b) built 2026-09-16T00:00:00Z" {
		t.Fatalf("unexpected summary: %q", got)
	}
	if !Committed() {
		t.Fatal("injected build should be Committed")
	}
}

func TestSummaryIgnoresMalformedBuildTime(t *testing.T) {
	Version, Commit, BuildTime = "v1.0.0", "", "not-a-time"
	defer func() { Version, Commit, BuildTime = "dev", "", "" }()
	if got := Summary(); got != "v1.0.0" {
		t.Fatalf("malformed build time should be omitted, got %q", got)
	}
}
