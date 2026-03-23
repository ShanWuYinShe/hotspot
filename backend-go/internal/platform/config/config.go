package config

import "os"

type Config struct {
	AppName            string
	Host               string
	Port               string
	PostgresDSN        string
	RedisDSN           string
	AuthTokenSecret    string
	SupportEmail       string
	PrivacyPolicyURL   string
	TermsOfServiceURL  string
	OfficialWebsiteURL string
	DefaultUserAvatar  string
	DefaultGroupAvatar string
}

func Load() Config {
	return Config{
		AppName:            envOrDefault("APP_NAME", "hotspot-backend-go"),
		Host:               envOrDefault("APP_HOST", "0.0.0.0"),
		Port:               envOrDefault("APP_PORT", "8081"),
		PostgresDSN:        os.Getenv("POSTGRES_DSN"),
		RedisDSN:           envOrDefault("REDIS_DSN", "redis://localhost:6379/0"),
		AuthTokenSecret:    envOrDefault("AUTH_TOKEN_SECRET", "dev-secret-change-me"),
		SupportEmail:       envOrDefault("APP_SUPPORT_EMAIL", "support@example.com"),
		PrivacyPolicyURL:   envOrDefault("APP_PRIVACY_POLICY_URL", ""),
		TermsOfServiceURL:  envOrDefault("APP_TERMS_OF_SERVICE_URL", ""),
		OfficialWebsiteURL: envOrDefault("APP_OFFICIAL_WEBSITE_URL", ""),
		DefaultUserAvatar:  envOrDefault("APP_DEFAULT_USER_AVATAR_URL", ""),
		DefaultGroupAvatar: envOrDefault("APP_DEFAULT_GROUP_AVATAR_URL", ""),
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
