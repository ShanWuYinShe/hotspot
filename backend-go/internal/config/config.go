package config

import (
	"encoding/json"
	"os"
)

// Config 定义 backend-go 服务运行所需的关键配置。
//
// 行业最佳实践：
// - 无论是正常运行还是测试，始终复用同一份 Config 结构；
// - 区分环境的方式只体现在“值的来源”（不同 env / 不同文件），而不是结构本身。
type Config struct {
	HTTPAddr    string `json:"http_addr"`
	PostgresURL string `json:"postgres_url"`
	RedisAddr   string `json:"redis_addr"`
	JWTSecret   string `json:"jwt_secret"`
}

// Load 从环境变量加载配置，必要字段为空时可以在上层进行校验或报错。
// 运行正常服务时仍推荐使用环境变量注入配置（12-factor 风格）。
func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		PostgresURL: os.Getenv("POSTGRES_URL"),
		RedisAddr:   getenv("REDIS_ADDR", ""),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
}

// LoadFromFile 从 JSON 配置文件加载 Config。
//
// 文件中的字段名与结构体 tag 对齐，例如：
//
//	{
//	  "http_addr": ":8080",
//	  "postgres_url": "postgres://...",
//	  "redis_addr": "localhost:6379",
//	  "jwt_secret": "..."
//	}
func LoadFromFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, err
	}

	// 与 Load 行为保持一致：必要时填充默认值。
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
