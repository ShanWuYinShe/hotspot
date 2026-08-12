package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// defaultBcryptCost is the OWASP-recommended minimum (≥12 as of 2023).
// Operators can override via BCRYPT_COST, subject to Validate() clamping.
const defaultBcryptCost = 12

type Config struct {
	AppName                string
	AppEnv                 string
	LogLevel               string
	Host                   string
	Port                   string
	PostgresDSN            string
	AuthTokenSecret        string
	SuperAdminPassword     string
	SupportEmail           string
	PrivacyPolicyURL       string
	TermsOfServiceURL      string
	OfficialWebsiteURL     string
	DefaultUserAvatar      string
	DefaultGroupAvatar     string
	TrustedProxyCIDRs      string
	S3Endpoint             string
	S3PublicEndpoint       string
	S3Region               string
	S3Bucket               string
	S3AccessKey            string
	S3SecretKey            string
	S3ForcePathStyle       bool
	S3PublicForcePathStyle bool
	RedisURL               string
	// ResumeTTLSeconds 控制 resume 增量恢复状态的保留秒数（spec L156：TTL ≈ 2×
	// ping_interval，默认 60）。连接断开后服务端在该窗口内可增量续传；超期回退全量。
	ResumeTTLSeconds int
	// BcryptCost controls the work factor for password hashing. Must be within
	// [bcrypt.MinCost, bcrypt.MaxCost]; Validate() rejects anything outside
	// that range. Defaults to defaultBcryptCost (OWASP minimum).
	BcryptCost int
}

func Load() Config {
	return Config{
		AppName:                envOrDefault("APP_NAME", "hotspot-backend-go"),
		AppEnv:                 envOrDefault("APP_ENV", "production"),
		LogLevel:               envOrDefault("LOG_LEVEL", "info"),
		Host:                   envOrDefault("APP_HOST", "0.0.0.0"),
		Port:                   envOrDefault("APP_PORT", "8081"),
		PostgresDSN:            os.Getenv("POSTGRES_DSN"),
		AuthTokenSecret:        envOrDefault("AUTH_TOKEN_SECRET", devDefaultAuthTokenSecret),
		SuperAdminPassword:     os.Getenv("SUPER_ADMIN_PASSWORD"),
		SupportEmail:           envOrDefault("APP_SUPPORT_EMAIL", "support@example.com"),
		PrivacyPolicyURL:       envOrDefault("APP_PRIVACY_POLICY_URL", ""),
		TermsOfServiceURL:      envOrDefault("APP_TERMS_OF_SERVICE_URL", ""),
		OfficialWebsiteURL:     envOrDefault("APP_OFFICIAL_WEBSITE_URL", ""),
		DefaultUserAvatar:      envOrDefault("APP_DEFAULT_USER_AVATAR_URL", ""),
		DefaultGroupAvatar:     envOrDefault("APP_DEFAULT_GROUP_AVATAR_URL", ""),
		TrustedProxyCIDRs:      envOrDefault("TRUSTED_PROXY_CIDRS", ""),
		S3Endpoint:             envOrDefault("S3_ENDPOINT", ""),
		S3PublicEndpoint:       envOrDefault("S3_PUBLIC_ENDPOINT", ""),
		S3Region:               envOrDefault("S3_REGION", "us-east-1"),
		S3Bucket:               envOrDefault("S3_BUCKET", ""),
		S3AccessKey:            envOrDefault("S3_ACCESS_KEY", ""),
		S3SecretKey:            envOrDefault("S3_SECRET_KEY", ""),
		S3ForcePathStyle:       envBoolOrDefault("S3_FORCE_PATH_STYLE", true),
		S3PublicForcePathStyle: envBoolOrDefault("S3_PUBLIC_FORCE_PATH_STYLE", true),
		RedisURL:               os.Getenv("REDIS_URL"),
		ResumeTTLSeconds:       envIntOrDefault("RESUME_TTL_SECONDS", 60),
		BcryptCost:             bcryptCostOrDefault("BCRYPT_COST", defaultBcryptCost),
	}
}

func (c Config) HTTPAddr() string {
	return c.Host + ":" + c.Port
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "TRUE" || value == "True"
}

// envIntOrDefault 解析正整数环境变量；空/非法/非正则返回 fallback。
func envIntOrDefault(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

// bcryptCostOrDefault parses BCRYPT_COST. An empty/unset env var yields the
// fallback; a malformed or out-of-range value also yields the fallback (the
// clamping is defensive — Validate() still enforces the bound at startup so
// a typo can't silently weaken hashing).
func bcryptCostOrDefault(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	cost, err := strconv.Atoi(raw)
	if err != nil || cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return fallback
	}
	return cost
}

// devDefaultAuthTokenSecret is the well-known key used only as a local-dev
// fallback. Validate() rejects it (and any empty value) in production so the
// process never starts with a secret an attacker could read from source.
const devDefaultAuthTokenSecret = "dev-secret-change-me"

// DevDefaultAuthTokenSecret exposes the dev fallback so other packages (e.g.
// auth.NewTokenManager) reference the same value instead of re-hardcoding it.
func DevDefaultAuthTokenSecret() string { return devDefaultAuthTokenSecret }

// DefaultBcryptCost exposes the OWASP-recommended bcrypt minimum so other
// packages (e.g. modules/auth) reference the same value instead of
// re-hardcoding it.
func DefaultBcryptCost() int { return defaultBcryptCost }

// knownWeakSecrets are values that must never sign production tokens because
// they appear in source control and are therefore publicly readable.
var knownWeakSecrets = map[string]struct{}{
	devDefaultAuthTokenSecret: {},
}

// Severity classifies how dangerous an insecure default is.
type Severity int

const (
	// SeverityWarning means the value is weak but the deployment can boot.
	SeverityWarning Severity = iota
	// SeverityCritical means the value is a publicly-known default; production
	// must refuse to boot when a critical default is detected.
	SeverityCritical
)

func (s Severity) String() string {
	if s == SeverityCritical {
		return "CRITICAL"
	}
	return "WARNING"
}

// DefaultCheck describes one (field, known-default-value, severity) rule that
// checkInsecureDefaults() evaluates. If the field's current value matches the
// sentinel, the rule fires and produces a DefaultFinding.
type DefaultCheck struct {
	Field     string   // human-readable field/env name shown in messages
	Value     string   // the value held in Config (already loaded)
	Sentinels []string // known insecure default values that match this field
	Severity  Severity
	// CombineWith optionally ANDs this rule with a sibling (e.g. require both
	// S3 access + secret to be minioadmin before flagging). When non-empty,
	// the rule only fires if every named sibling field also holds one of its
	// own sentinels at the same severity. Siblings are resolved against the
	// same registry by field name.
	CombineWith []string
}

// DefaultFinding is one detected insecure default. checkInsecureDefaults
// returns the full list; the caller decides whether critical findings abort.
type DefaultFinding struct {
	Field    string
	Severity Severity
	Message  string
}

// insecureDefaultRegistry is the table of (field -> known-bad values) the
// guardrail walks at startup. Adding a new default to block is a one-line
// change here. Sentinel values come from the repo's own .env / .env.example
// files (and the legacy Python backend/.env) so an operator who copies an
// example verbatim is caught.
func (c Config) insecureDefaultRegistry() []DefaultCheck {
	return []DefaultCheck{
		{
			Field:     "AUTH_TOKEN_SECRET",
			Value:     c.AuthTokenSecret,
			Sentinels: []string{devDefaultAuthTokenSecret, "change-me", "dev-secret-key-change-in-production", "your-super-secret-jwt-key-change-this-in-production", "dev-only-do-not-use-in-production-please-change-to-random-48-bytes"},
			Severity:  SeverityCritical,
		},
		{
			Field:     "SUPER_ADMIN_PASSWORD",
			Value:     c.SuperAdminPassword,
			Sentinels: []string{"liyou19890317", "change-this-strong-password", "change-me"},
			Severity:  SeverityCritical,
		},
		{
			// SeaweedFS/MinIO ships with minioadmin/minioadmin. Only fire when
			// BOTH are default — single-side rotations still let boot proceed
			// (the rotated side is the meaningful change).
			Field:       "S3_ACCESS_KEY",
			Value:       c.S3AccessKey,
			Sentinels:   []string{"minioadmin"},
			Severity:    SeverityCritical,
			CombineWith: []string{"S3_SECRET_KEY"},
		},
		{
			Field:       "S3_SECRET_KEY",
			Value:       c.S3SecretKey,
			Sentinels:   []string{"minioadmin", "change-me"},
			Severity:    SeverityCritical,
			CombineWith: []string{"S3_ACCESS_KEY"},
		},
	}
}

// checkInsecureDefaults walks the default registry and returns every finding
// that fires for the current config. Pure function — no I/O, no logging — so
// it is trivially unit-testable. Validate() uses the result to decide whether
// to abort (production) and main.go logs the warnings in dev.
func (c Config) checkInsecureDefaults() []DefaultFinding {
	rules := c.insecureDefaultRegistry()
	byField := make(map[string]DefaultCheck, len(rules))
	for _, r := range rules {
		byField[r.Field] = r
	}

	var findings []DefaultFinding
	for _, r := range rules {
		if !containsString(r.Sentinels, r.Value) {
			continue
		}
		if len(r.CombineWith) > 0 {
			combined := true
			for _, siblingName := range r.CombineWith {
				sibling, ok := byField[siblingName]
				if !ok || !containsString(sibling.Sentinels, sibling.Value) {
					combined = false
					break
				}
			}
			if !combined {
				continue
			}
		}
		findings = append(findings, DefaultFinding{
			Field:    r.Field,
			Severity: r.Severity,
			Message: fmt.Sprintf(
				"%s is set to a known default/placeholder value (%q) — this value is checked into source control and must be changed before serving real traffic",
				r.Field, r.Value,
			),
		})
	}
	return findings
}

// hasCritical reports whether any finding is SeverityCritical. Helper for
// callers (Validate, main) that only care about the abort decision.
func (c Config) hasCritical(findings []DefaultFinding) bool {
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// CheckInsecureDefaultsPublic exposes the default-scan result so callers
// outside the config package (e.g. main, an admin debug endpoint) can surface
// the same findings Validate() uses. It is the public wrapper around the
// internal checkInsecureDefaults().
func (c Config) CheckInsecureDefaultsPublic() []DefaultFinding {
	return c.checkInsecureDefaults()
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// Validate enforces configuration invariants required for a safe production
// deployment. It is intentionally lenient in development so local workflows
// keep working without setting every env var. Call it once at startup; a
// returned error means the process must refuse to serve traffic.
//
// In production (or when APP_ENV is unset — see isProductionLike), it rejects:
//   - empty / weak / short AUTH_TOKEN_SECRET
//   - any insecure default from checkInsecureDefaults() (registry-driven)
//   - empty PostgresDSN / RedisURL (a prod server with no DB is unsafe)
func (c Config) Validate() error {
	if !c.isProductionLike() {
		return nil
	}
	if c.BcryptCost < bcrypt.MinCost || c.BcryptCost > bcrypt.MaxCost {
		return fmt.Errorf("BCRYPT_COST must be in [%d, %d] (got %d); see bcrypt cost documentation", bcrypt.MinCost, bcrypt.MaxCost, c.BcryptCost)
	}
	if c.BcryptCost < defaultBcryptCost {
		return fmt.Errorf("BCRYPT_COST must be at least %d in production (got %d); lower values are below the OWASP-recommended minimum", defaultBcryptCost, c.BcryptCost)
	}
	if c.AuthTokenSecret == "" {
		return errors.New("AUTH_TOKEN_SECRET must be set in production (refusing to start with no token secret)")
	}
	if _, weak := knownWeakSecrets[c.AuthTokenSecret]; weak {
		return fmt.Errorf("AUTH_TOKEN_SECRET must be changed from the built-in dev default %q (refusing to start with a publicly-known token secret)", devDefaultAuthTokenSecret)
	}
	if len(c.AuthTokenSecret) < 32 {
		return fmt.Errorf("AUTH_TOKEN_SECRET must be at least 32 bytes in production (got %d); use a high-entropy random value", len(c.AuthTokenSecret))
	}
	if c.PostgresDSN == "" {
		return errors.New("POSTGRES_DSN must be set in production (refusing to start with no database)")
	}
	if c.RedisURL == "" {
		return errors.New("REDIS_URL must be set in production (refusing to start with no Redis — WebSocket fanout would be unsafe across instances)")
	}
	if findings := c.checkInsecureDefaults(); c.hasCritical(findings) {
		return fmt.Errorf("insecure default configuration detected:\n%s", formatFindings(findings))
	}
	return nil
}

// formatFindings renders findings as a bulleted, severity-prefixed block for
// inclusion in error messages and log output.
func formatFindings(findings []DefaultFinding) string {
	if len(findings) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "  [%s] %s: %s\n", f.Severity, f.Field, f.Message)
	}
	return b.String()
}

// isProductionLike reports whether the environment is not explicitly a
// development or test environment. Anything unset defaults to "production"
// (see Load), so an operator who forgets APP_ENV gets the safe (strict) path.
func (c Config) isProductionLike() bool {
	env := strings.ToLower(strings.TrimSpace(c.AppEnv))
	return env == "" || env == "production" || env == "prod" || env == "staging" || env == "stage"
}
