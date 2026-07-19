package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idgen"

	"golang.org/x/crypto/bcrypt"
)

const verificationCodeDigits = 6

// hashCost is the bcrypt cost used by hashPassword. It defaults to the OWASP
// minimum (config.DefaultBcryptCost) and can be reconfigured once at startup
// via ConfigureHashingCost. Stored in an atomic so a runtime reconfigure (or
// a test) doesn't race with concurrent Register/ChangePassword calls.
var hashCost atomic.Int32

func init() {
	hashCost.Store(int32(config.DefaultBcryptCost()))
}

// DefaultBcryptCost exposes the configured default so tests and tooling can
// reference the same value without re-hardcoding it.
func DefaultBcryptCost() int { return int(hashCost.Load()) }

// ConfigureHashingCost sets the bcrypt work factor used by hashPassword. It
// is intended to be called once at module construction (see module.go) from
// the validated Config.BcryptCost. Values outside [bcrypt.MinCost,
// bcrypt.MaxCost] are silently clamped into range so a misconfigured caller
// can't crash the process at hash time.
func ConfigureHashingCost(cost int) {
	if cost < bcrypt.MinCost {
		cost = bcrypt.MinCost
	}
	if cost > bcrypt.MaxCost {
		cost = bcrypt.MaxCost
	}
	hashCost.Store(int32(cost))
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), int(hashCost.Load()))
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyPassword(storedHash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}

// generateUserID delegates to the shared idgen package so that user IDs and
// all other entity IDs share one collision-resistant generator.
func generateUserID() (string, error) {
	return idgen.WithPrefix("user_")
}

func generateVerificationCode() (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(verificationCodeDigits), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", verificationCodeDigits, n.Int64()), nil
}
