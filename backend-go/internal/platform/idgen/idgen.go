// Package idgen produces unprefixed random identifier strings used as the
// body of prefixed primary keys (e.g. "moment_<id>", "user_<id>").
//
// The previous implementation in the auth/moments/groups/comments modules
// derived entity IDs from time.Now().UnixNano(). Under concurrent inserts
// that approach can produce identical IDs in the same nanosecond and cause
// primary-key collisions. idgen instead draws 96 bits of entropy from
// crypto/rand and hex-encodes it (24 ASCII characters), matching the entropy
// of the existing user-ID generator while remaining prefix-agnostic.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// entropyBytes controls how many bytes of crypto-random data back each ID.
// 12 bytes = 96 bits of entropy, the same amount the auth module used for
// user IDs. With 96 bits the probability of a collision reaches 50% only
// after ~2^48 IDs per prefix, far beyond any realistic write volume.
const entropyBytes = 12

// New returns a fresh hex-encoded random identifier (no prefix). Callers
// prepend their own type prefix (e.g. "moment_", "group_").
func New() (string, error) {
	buf := make([]byte, entropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// MustNew is the panic-on-error variant of New, for call sites where a
// crypto/rand failure is unrecoverable (none in practice). Prefer New in
// production code paths.
func MustNew() string {
	id, err := New()
	if err != nil {
		panic(err)
	}
	return id
}

// WithPrefix returns "<prefix><random id>". It is a small convenience over
// calling New and concatenating manually; use it to keep the prefix
// convention explicit at each call site.
func WithPrefix(prefix string) (string, error) {
	id, err := New()
	if err != nil {
		return "", err
	}
	return prefix + id, nil
}

// MustWithPrefix is the panic-on-error variant of WithPrefix, for struct
// literal initialization where returning an error is impossible.
func MustWithPrefix(prefix string) string {
	id, err := WithPrefix(prefix)
	if err != nil {
		panic(err)
	}
	return id
}
