package groups

import (
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/notify/notifytest"
)

// NewService 是测试专用便捷构造（生产装配显式注入依赖；内存仓储实现已迁入
// 测试构建 / notifytest，P0-2）。
func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), notify.NewServiceWithRepo(notifytest.New()), nil)
}

// NewServiceWithRepo 是测试专用便捷构造。
func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDeps(repo, notify.NewServiceWithRepo(notifytest.New()), nil)
}
