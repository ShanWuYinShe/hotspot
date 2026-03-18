package config

import "os"

// Config 定义 backend-go 服务运行所需的关键配置。
// 后续如有需要可在此处扩展字段，并在 Load 中补充读取逻辑。
type Config struct {
	HTTPAddr    string
	PostgresURL string
	RedisAddr   string
	JWTSecret   string
}

// Load 从环境变量加载配置，必要字段为空时可以在上层进行校验或报错。
func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		PostgresURL: os.Getenv("POSTGRES_URL"),
		RedisAddr:   getenv("REDIS_ADDR", ""),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
