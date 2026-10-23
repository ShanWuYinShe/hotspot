package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
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

// timingEqualizationHash 是「账号不存在」分支做等长 bcrypt 比对用的固定哈希
// （审查 P2-5：时延侧信道）。响应体已防枚举（凭据错误与账号不存在同响应），
// 但账号不存在时若直接返回，省掉约 100ms+ 的 bcrypt 计算，攻击者可据耗时
// 探测账号是否存在。该哈希按当前 cost 惰性生成并缓存（cost 只在启动期
// 配置，常量成本下缓存安全；cost 变更时 nextCost 标记重建）。
var (
	timingHashMu    sync.Mutex
	timingHashValue string
	timingHashCost  int32
	timingHashDummy = "timing-equalization-dummy-password"
)

// equalizeVerifyTiming 在「账号不存在」分支执行一次与真实校验同代价的 bcrypt
// 比对，抹平时延差异。生成哈希失败时静默跳过（不影响登录语义）。
func equalizeVerifyTiming(password string) {
	cost := int32(hashCost.Load())
	timingHashMu.Lock()
	if timingHashValue == "" || timingHashCost != cost {
		hashed, err := bcrypt.GenerateFromPassword([]byte(timingHashDummy), int(cost))
		if err != nil {
			timingHashMu.Unlock()
			return
		}
		timingHashValue = string(hashed)
		timingHashCost = cost
	}
	stored := timingHashValue
	timingHashMu.Unlock()
	_ = bcrypt.CompareHashAndPassword([]byte(stored), []byte(password))
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
