package notify

// NewService 是测试专用便捷构造（生产装配显式注入 Repository；内存实现已迁入
// 测试构建，P0-2）。
func NewService() *Service { return NewServiceWithRepo(NewInMemoryRepository()) }
