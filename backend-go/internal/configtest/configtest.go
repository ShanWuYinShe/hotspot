package configtest

import (
	"os"
	"testing"

	"backend-go/internal/config"
)

// Load 为测试场景加载 backend-go 配置。
//
// 行为约定：
//   - 正常运行与测试场景共享同一份 Config 结构（config.Config）；
//   - 测试通过“单独的配置文件”提供不同的值，而不是引入另一套 key；
//   - 若测试配置文件不存在或无法解析，相关测试应显式 Skip，避免误连开发/生产库。
//
// 使用方式：
//   - 默认从项目根目录下的 "config.test.json" 加载；
//   - 也可以通过环境变量 BACKEND_GO_TEST_CONFIG_FILE 显式指定测试配置文件路径；
//   - 配置文件字段与 config.Config 的 JSON tag 一致：
//     { "http_addr": "...", "postgres_url": "...", "redis_addr": "...", "jwt_secret": "..." }。
func Load(t testing.TB) config.Config {
	t.Helper()

	path := os.Getenv("BACKEND_GO_TEST_CONFIG_FILE")
	if path == "" {
		path = "config.test.json"
	}

	cfg, err := config.LoadFromFile(path)
	if err != nil {
		t.Skipf("cannot load test config file %s: %v; skipping tests that require real backend config", path, err)
	}

	return cfg
}
