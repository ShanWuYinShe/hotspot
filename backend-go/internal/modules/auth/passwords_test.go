package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestHashPasswordUsesConfiguredCost verifies that ConfigureHashingCost
// actually changes the work factor exercised by hashPassword. Hashing at
// cost 4 (MinCost) keeps the test fast (<10ms) while still proving the
// override took effect via bcrypt.Cost on the resulting hash.
func TestHashPasswordUsesConfiguredCost(t *testing.T) {
	t.Cleanup(func() { ConfigureHashingCost(12) }) // restore OWASP default
	ConfigureHashingCost(bcrypt.MinCost)
	hashed, err := hashPassword("hunter2")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hashed))
	if err != nil {
		t.Fatalf("bcrypt.Cost failed: %v", err)
	}
	if cost != bcrypt.MinCost {
		t.Fatalf("expected cost %d from configured override, got %d", bcrypt.MinCost, cost)
	}
}

// TestConfigureHashingCostClamps verifies that out-of-range costs get
// clamped into bcrypt's supported window. We check the *configured* value
// via DefaultBcryptCost() rather than invoking hashPassword, because
// running bcrypt at MaxCost (31) takes minutes and would time out the
// test — the clamp is what we care about, not the hashing itself.
func TestConfigureHashingCostClamps(t *testing.T) {
	t.Cleanup(func() { ConfigureHashingCost(12) })
	cases := []struct {
		in   int
		want int
	}{
		{in: 0, want: bcrypt.MinCost},
		{in: -1, want: bcrypt.MinCost},
		{in: 3, want: bcrypt.MinCost},
		{in: bcrypt.MinCost, want: bcrypt.MinCost},
		{in: bcrypt.MaxCost, want: bcrypt.MaxCost},
		{in: bcrypt.MaxCost + 1, want: bcrypt.MaxCost},
		{in: 100, want: bcrypt.MaxCost},
	}
	for _, c := range cases {
		ConfigureHashingCost(c.in)
		if got := DefaultBcryptCost(); got != c.want {
			t.Fatalf("ConfigureHashingCost(%d): expected clamp to %d, got %d", c.in, c.want, got)
		}
	}
}

// TestDefaultBcryptCostMatchesOWASPMinimum documents the invariant that the
// package default is the OWASP-recommended floor.
func TestDefaultBcryptCostMatchesOWASPMinimum(t *testing.T) {
	if got := DefaultBcryptCost(); got < 12 {
		t.Fatalf("DefaultBcryptCost must be >= OWASP minimum 12, got %d", got)
	}
}

// TestHashPasswordStillProducesVerifiableHash is a sanity check that the
// atomic-introduction didn't break the verify path.
func TestHashPasswordStillProducesVerifiableHash(t *testing.T) {
	t.Cleanup(func() { ConfigureHashingCost(12) })
	ConfigureHashingCost(bcrypt.MinCost)
	hashed, err := hashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}
	if !verifyPassword(hashed, "correct-horse-battery-staple") {
		t.Fatal("verifyPassword rejected the password it just hashed")
	}
	if verifyPassword(hashed, "wrong") {
		t.Fatal("verifyPassword accepted wrong password")
	}
}
