package privatemessages

import (
	"context"

	"hotspot/backend-go/internal/modules/users"
)

// permissiveUsers 是 privatemessages.userReader 的最小测试桩（users 内存仓储
// 已迁入其模块测试构建，P0-2）：任意用户均视为存在、默认 everyone 私聊权限、
// 无拉黑/关注关系，保持同包用例原有的宽松发送语义。
type permissiveUsers struct{}

func (permissiveUsers) OpenTheDoor(_ context.Context, userID string) (users.UserItem, bool, error) {
	return users.UserItem{User: users.UserInfo{UserID: userID, Nickname: userID}}, true, nil
}

func (permissiveUsers) BatchGet(_ context.Context, userIDs []string) (map[string]users.UserInfo, error) {
	out := make(map[string]users.UserInfo, len(userIDs))
	for _, id := range userIDs {
		out[id] = users.UserInfo{UserID: id, Nickname: id}
	}
	return out, nil
}

func (permissiveUsers) SettingsFor(context.Context, string) (users.UserSettings, bool, error) {
	return users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}, true, nil
}

func (permissiveUsers) BlockStateBetween(context.Context, string, string) (bool, bool, error) {
	return false, false, nil
}

func (permissiveUsers) Relationship(context.Context, string, string) (users.Relationship, error) {
	return users.Relationship{}, nil
}

func (permissiveUsers) RelationshipBatch(_ context.Context, _ string, targetUserIDs []string) (map[string]users.Relationship, error) {
	out := make(map[string]users.Relationship, len(targetUserIDs))
	for _, id := range targetUserIDs {
		out[id] = users.Relationship{}
	}
	return out, nil
}

// 测试便捷构造函数：生产装配完全显式（bootstrap 注入 userReader），以下仅供
// 同包单测构造带内存仓储/用户读取器的服务。放在 _test.go 避免生产代码引用
// 其它模块的内存仓储（架构审计 P0-2 / archtest 跨模块仓储构造约束）。
func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), permissiveUsers{})
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDeps(repo, permissiveUsers{})
}

func NewRuntimeWithRepo(repo Repository) *Runtime {
	service := NewServiceWithRepo(repo)
	return &Runtime{
		handler:     NewHandler(service),
		readService: service,
	}
}
