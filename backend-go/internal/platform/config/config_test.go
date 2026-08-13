package config

import (
	"os"
	"testing"
)

func TestConfigHTTPAddr(t *testing.T) {
	cfg := Config{Host: "127.0.0.1", Port: "9000"}
	if got := cfg.HTTPAddr(); got != "127.0.0.1:9000" {
		t.Fatalf("unexpected http addr: %s", got)
	}
}

func TestConfigCarriesDatasourceValues(t *testing.T) {
	cfg := Config{
		PostgresDSN: "postgres://test-db/hotspot_test",
	}
	if cfg.PostgresDSN != "postgres://test-db/hotspot_test" {
		t.Fatalf("unexpected postgres dsn: %s", cfg.PostgresDSN)
	}
}

func TestLoadLeavesPostgresDSNEmptyWhenUnset(t *testing.T) {
	oldPostgres := os.Getenv("POSTGRES_DSN")
	oldTrustedProxyCIDRs := os.Getenv("TRUSTED_PROXY_CIDRS")
	defer func() {
		if oldPostgres == "" {
			_ = os.Unsetenv("POSTGRES_DSN")
		} else {
			_ = os.Setenv("POSTGRES_DSN", oldPostgres)
		}
		if oldTrustedProxyCIDRs == "" {
			_ = os.Unsetenv("TRUSTED_PROXY_CIDRs")
		} else {
			_ = os.Setenv("TRUSTED_PROXY_CIDRs", oldTrustedProxyCIDRs)
		}
	}()

	_ = os.Unsetenv("POSTGRES_DSN")
	_ = os.Unsetenv("TRUSTED_PROXY_CIDRs")

	cfg := Load()
	if cfg.PostgresDSN != "" {
		t.Fatalf("expected empty postgres dsn, got %q", cfg.PostgresDSN)
	}
	if cfg.TrustedProxyCIDRs != "" {
		t.Fatalf("expected empty trusted proxy cidrs, got %q", cfg.TrustedProxyCIDRs)
	}
}

func TestLoadCarriesTrustedProxyCIDRs(t *testing.T) {
	oldValue := os.Getenv("TRUSTED_PROXY_CIDRS")
	defer func() {
		if oldValue == "" {
			_ = os.Unsetenv("TRUSTED_PROXY_CIDRS")
		} else {
			_ = os.Setenv("TRUSTED_PROXY_CIDRS", oldValue)
		}
	}()

	_ = os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,192.168.0.0/16")
	cfg := Load()
	if cfg.TrustedProxyCIDRs != "10.0.0.0/8,192.168.0.0/16" {
		t.Fatalf("unexpected trusted proxy cidrs: %q", cfg.TrustedProxyCIDRs)
	}
}

func TestLoadCarriesS3PublicEndpointSettings(t *testing.T) {
	oldEndpoint := os.Getenv("S3_PUBLIC_ENDPOINT")
	oldForcePathStyle := os.Getenv("S3_PUBLIC_FORCE_PATH_STYLE")
	defer func() {
		if oldEndpoint == "" {
			_ = os.Unsetenv("S3_PUBLIC_ENDPOINT")
		} else {
			_ = os.Setenv("S3_PUBLIC_ENDPOINT", oldEndpoint)
		}
		if oldForcePathStyle == "" {
			_ = os.Unsetenv("S3_PUBLIC_FORCE_PATH_STYLE")
		} else {
			_ = os.Setenv("S3_PUBLIC_FORCE_PATH_STYLE", oldForcePathStyle)
		}
	}()

	_ = os.Setenv("S3_PUBLIC_ENDPOINT", "http://localhost:8333")
	_ = os.Setenv("S3_PUBLIC_FORCE_PATH_STYLE", "false")

	cfg := Load()
	if cfg.S3PublicEndpoint != "http://localhost:8333" {
		t.Fatalf("unexpected s3 public endpoint: %q", cfg.S3PublicEndpoint)
	}
	if cfg.S3PublicForcePathStyle {
		t.Fatal("expected s3 public force path style to be false")
	}
}

func TestValidateRejectsDefaultSecretInProduction(t *testing.T) {
	// Load() falls back to the built-in dev secret when AUTH_TOKEN_SECRET is
	// unset; Validate must refuse to ship that in production.
	t.Setenv("AUTH_TOKEN_SECRET", "")
	cfg := Config{AppEnv: "production", AuthTokenSecret: devDefaultAuthTokenSecret}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when production uses the dev default secret")
	}
}

// --- insecure-defaults registry tests (Batch C guardrail) ---

// prodBootable returns a Config with all production invariants satisfied so
// individual tests can introduce exactly one weak value at a time.
func prodBootable() Config {
	return Config{
		AppEnv:             "production",
		AuthTokenSecret:    stringOfLen(48),
		PostgresDSN:        "postgres://postgres:realpw@db:5432/hotspot",
		RedisURL:           "redis://redis:6379/0",
		S3AccessKey:        "real-access-key",
		S3SecretKey:        "real-secret-key",
		BcryptCost:         defaultBcryptCost,
	}
}

func TestCheckInsecureDefaultsClean(t *testing.T) {
	cfg := prodBootable()
	if got := cfg.checkInsecureDefaults(); len(got) != 0 {
		t.Fatalf("expected no findings for clean config, got: %+v", got)
	}
}

func TestCheckInsecureDefaultsDetectsDevAuthTokenSecret(t *testing.T) {
	cfg := prodBootable()
	cfg.AuthTokenSecret = devDefaultAuthTokenSecret
	findings := cfg.checkInsecureDefaults()
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for dev auth secret, got %+v", findings)
	}
	if findings[0].Field != "AUTH_TOKEN_SECRET" || findings[0].Severity != SeverityCritical {
		t.Fatalf("unexpected finding: %+v", findings[0])
	}
}

func TestCheckInsecureDefaultsDetectsLegacyJWTDefault(t *testing.T) {
	// Value copied from the legacy backend/.env — must be blocked even though
	// it is not the Go-side dev default.
	cfg := prodBootable()
	cfg.AuthTokenSecret = "dev-secret-key-change-in-production"
	if findings := cfg.checkInsecureDefaults(); !cfg.hasCritical(findings) {
		t.Fatalf("expected critical finding for legacy JWT default, got %+v", findings)
	}
}

func TestCheckInsecureDefaultsRequiresBothS3Defaults(t *testing.T) {
	// Rotating just one side must NOT fire — single-side rotation is the
	// meaningful change and shouldn't be blocked.
	cfg := prodBootable()
	cfg.S3AccessKey = "minioadmin" // secret still strong
	if findings := cfg.checkInsecureDefaults(); len(findings) != 0 {
		t.Fatalf("expected no finding when only S3 access key is default, got %+v", findings)
	}
}

func TestCheckInsecureDefaultsDetectsBothS3Defaults(t *testing.T) {
	cfg := prodBootable()
	cfg.S3AccessKey = "minioadmin"
	cfg.S3SecretKey = "minioadmin"
	findings := cfg.checkInsecureDefaults()
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (access + secret) when both default, got %+v", findings)
	}
	for _, f := range findings {
		if f.Severity != SeverityCritical {
			t.Fatalf("expected critical severity, got %+v", f)
		}
		if f.Field != "S3_ACCESS_KEY" && f.Field != "S3_SECRET_KEY" {
			t.Fatalf("unexpected field: %+v", f)
		}
	}
}

func TestCheckInsecureDefaultsAllowsDevModeBoot(t *testing.T) {
	// Dev mode: defaults are present but checkInsecureDefaults still returns
	// the list (so main can warn). Validate() itself is lenient in dev.
	cfg := Config{
		AppEnv:             "development",
		AuthTokenSecret:    devDefaultAuthTokenSecret,
		S3AccessKey:        "minioadmin",
		S3SecretKey:        "minioadmin",
	}
	findings := cfg.checkInsecureDefaults()
	if len(findings) == 0 {
		t.Fatal("expected findings to be reported in dev mode too")
	}
	// And boot is allowed:
	if err := cfg.Validate(); err != nil {
		t.Fatalf("dev mode should allow boot despite defaults, got %v", err)
	}
}

func TestValidateRejectsEmptyPostgresDSNInProduction(t *testing.T) {
	cfg := prodBootable()
	cfg.PostgresDSN = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when PostgresDSN empty in production")
	}
}

func TestValidateRejectsEmptyRedisURLInProduction(t *testing.T) {
	cfg := prodBootable()
	cfg.RedisURL = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when RedisURL empty in production")
	}
}

func TestValidateRejectsAnyCriticalDefaultInProduction(t *testing.T) {
	cases := map[string]func(Config) Config{
		"dev auth secret":      func(c Config) Config { c.AuthTokenSecret = devDefaultAuthTokenSecret; return c },
		"both s3 default":      func(c Config) Config { c.S3AccessKey = "minioadmin"; c.S3SecretKey = "minioadmin"; return c },
		"legacy jwt default":   func(c Config) Config { c.AuthTokenSecret = "dev-secret-key-change-in-production"; return c },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := mutate(prodBootable())
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected production to reject %q", name)
			}
		})
	}
}

func TestValidateAcceptsCleanProductionConfig(t *testing.T) {
	cfg := prodBootable()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected clean prod config to validate, got %v", err)
	}
}

func TestValidateRejectsEmptySecretInProduction(t *testing.T) {
	cfg := Config{AppEnv: "production", AuthTokenSecret: ""}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when production secret is empty")
	}
}

func TestValidateRejectsShortSecretInProduction(t *testing.T) {
	cfg := Config{AppEnv: "production", AuthTokenSecret: "short-but-nondefault-key"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when production secret is shorter than 32 bytes")
	}
}

func TestValidateAcceptsStrongSecretInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:          "production",
		AuthTokenSecret: stringOfLen(48),
		PostgresDSN:     "postgres://postgres:pw@db:5432/hotspot",
		RedisURL:        "redis://redis:6379/0",
		BcryptCost:      defaultBcryptCost,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for strong production secret, got %v", err)
	}
}

func TestValidateAcceptsDevDefaultInDevelopment(t *testing.T) {
	// Local dev must keep working without forcing operators to set a secret.
	cfg := Config{AppEnv: "development", AuthTokenSecret: devDefaultAuthTokenSecret}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error in development, got %v", err)
	}
}

func TestValidateTreatsUnsetEnvAsProduction(t *testing.T) {
	// An operator who forgets APP_ENV gets the strict (safe) path.
	cfg := Config{AppEnv: "", AuthTokenSecret: devDefaultAuthTokenSecret}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when AppEnv is unset (defaults to production semantics)")
	}
}

func TestValidateAcceptsStagingWithStrongSecret(t *testing.T) {
	cfg := Config{
		AppEnv:          "staging",
		AuthTokenSecret: stringOfLen(40),
		PostgresDSN:     "postgres://postgres:pw@db:5432/hotspot",
		RedisURL:        "redis://redis:6379/0",
		BcryptCost:      defaultBcryptCost,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for staging with strong secret, got %v", err)
	}
}

func stringOfLen(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// --- bcrypt cost (Batch B hardening) ---

func TestBcryptCostOrDefaultReturnsDefaultWhenUnset(t *testing.T) {
	t.Setenv("BCRYPT_COST", "")
	if got := bcryptCostOrDefault("BCRYPT_COST", defaultBcryptCost); got != defaultBcryptCost {
		t.Fatalf("expected default %d when env unset, got %d", defaultBcryptCost, got)
	}
}

func TestBcryptCostOrDefaultParsesValidValue(t *testing.T) {
	t.Setenv("BCRYPT_COST", "14")
	if got := bcryptCostOrDefault("BCRYPT_COST", defaultBcryptCost); got != 14 {
		t.Fatalf("expected 14 from env, got %d", got)
	}
}

func TestBcryptCostOrDefaultFallsBackOnOutOfRange(t *testing.T) {
	cases := []string{"0", "3", "32", "100", "not-a-number", "-5"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("BCRYPT_COST", raw)
			if got := bcryptCostOrDefault("BCRYPT_COST", defaultBcryptCost); got != defaultBcryptCost {
				t.Fatalf("expected fallback for invalid value %q, got %d", raw, got)
			}
		})
	}
}

func TestValidateRejectsBcryptCostBelowOWASPMinimum(t *testing.T) {
	cfg := prodBootable()
	cfg.BcryptCost = 10 // bcrypt.DefaultCost, but below OWASP minimum of 12
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when BcryptCost is below OWASP minimum in production")
	}
}

func TestValidateRejectsBcryptCostOutOfRange(t *testing.T) {
	cfg := prodBootable()
	cfg.BcryptCost = 99
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when BcryptCost is out of bcrypt range")
	}
}

func TestValidateAcceptsOWASPMinimumBcryptCost(t *testing.T) {
	cfg := prodBootable()
	cfg.BcryptCost = defaultBcryptCost
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected OWASP minimum cost to validate, got %v", err)
	}
}
