package moments

import (
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/notify/notifytest"
)

// NewServiceWithRepo 是测试专用便捷构造（生产装配显式注入依赖；notify 内存
// 仓储实现已迁入 notifytest 测试包，P0-2）。
func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notify.NewServiceWithRepo(notifytest.New()), nil, nil, nil, nil)
}
