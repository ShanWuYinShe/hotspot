package privatemessages

import "hotspot/backend-go/internal/modules/users"

// 测试便捷构造函数：生产装配完全显式（bootstrap 注入 userReader），以下仅供
// 同包单测构造带内存仓储/用户读取器的服务。放在 _test.go 避免生产代码引用
// 其它模块的内存仓储（架构审计 P0-2 / archtest 跨模块仓储构造约束）。
func NewService() *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(NewInMemoryRepository(), users.NewServiceWithRepo(userRepo))
}

func NewServiceWithRepo(repo Repository) *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(repo, users.NewServiceWithRepo(userRepo))
}

func NewRuntimeWithRepo(repo Repository) *Runtime {
	service := NewServiceWithRepo(repo)
	return &Runtime{
		handler:     NewHandler(service),
		readService: service,
	}
}
