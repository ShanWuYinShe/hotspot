package idgen

import (
	"regexp"
	"testing"
)

// 24 hex（96 bit 熵）格式与唯一性冒烟：覆盖 New/WithPrefix/MustWithPrefix。
func TestNewFormatAndUniqueness(t *testing.T) {
	format := regexp.MustCompile(`^[0-9a-f]{24}$`)
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id, err := New()
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if !format.MatchString(id) {
			t.Fatalf("unexpected id format: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id within 1000 draws: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestWithPrefixPrepends(t *testing.T) {
	id, err := WithPrefix("moment_")
	if err != nil {
		t.Fatalf("WithPrefix: %v", err)
	}
	if len(id) != len("moment_")+24 || id[:7] != "moment_" {
		t.Fatalf("unexpected prefixed id: %q", id)
	}
	if MustWithPrefix("user_")[:5] != "user_" {
		t.Fatal("MustWithPrefix should keep prefix")
	}
}
