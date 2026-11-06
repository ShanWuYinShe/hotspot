package system

import "hotspot/backend-go/internal/platform/appconfig"

// 测试专用便捷构造函数：生产装配全部显式依赖；内存仓储实现已迁入测试构建。

func NewService(appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appConfig))
}
