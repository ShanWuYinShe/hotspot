package config

import "os"

type Config struct {
	AppName                string
	AppEnv                 string
	LogLevel               string
	Host                   string
	Port                   string
	PostgresDSN            string
	AuthTokenSecret        string
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
}

func Load() Config {
	return Config{
		AppName:                envOrDefault("APP_NAME", "hotspot-backend-go"),
		AppEnv:                 envOrDefault("APP_ENV", "production"),
		LogLevel:               envOrDefault("LOG_LEVEL", "info"),
		Host:                   envOrDefault("APP_HOST", "0.0.0.0"),
		Port:                   envOrDefault("APP_PORT", "8081"),
		PostgresDSN:            os.Getenv("POSTGRES_DSN"),
		AuthTokenSecret:        envOrDefault("AUTH_TOKEN_SECRET", "dev-secret-change-me"),
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
