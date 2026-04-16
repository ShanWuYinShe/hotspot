package slicex

import "testing"

func TestLimitCopyReturnsOriginalWhenLimitNonPositive(t *testing.T) {
	items := []int{1, 2, 3}
	got := LimitCopy(items, 0)
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestLimitCopyReturnsOriginalWhenShortEnough(t *testing.T) {
	items := []int{1, 2}
	got := LimitCopy(items, 5)
	if len(got) != 2 || got[1] != 2 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestLimitCopyTruncatesAndCopies(t *testing.T) {
	items := []int{1, 2, 3}
	got := LimitCopy(items, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("unexpected result: %#v", got)
	}
	got[0] = 99
	if items[0] != 1 {
		t.Fatalf("expected copy isolation, original mutated: %#v", items)
	}
}
