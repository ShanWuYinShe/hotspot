package groups

import "hotspot/backend-go/internal/modules/notify"

// NewService 是测试专用便捷构造（生产装配显式注入依赖；内存仓储实现已迁入
// 测试构建，P0-2）。
func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), notify.NewService(), nil)
}
