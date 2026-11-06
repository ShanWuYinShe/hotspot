package reports

// 测试专用便捷构造函数：生产装配全部显式依赖；内存仓储实现已迁入测试构建
//（架构审计 P0-2：内存仓储不再计入生产代码）。

func NewService() *Service { return NewServiceWithRepo(NewInMemoryRepository()) }
