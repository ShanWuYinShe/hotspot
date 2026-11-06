package privatechats

import (
	"context"

	"hotspot/backend-go/internal/modules/users"
)

// fakeUsers 同时实现 privatechats.userReader 与 privatemessages.userReader：
// users 内存仓储实现已迁入其模块测试构建（P0-2），privatechats 跨模块用例
// 只需要「目标用户存在/设置/关系」的可控读取语义。
type fakeUsers struct {
	items    map[string]users.UserItem
	settings map[string]users.UserSettings
	relation map[string]users.Relationship // "viewer|target"
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{
		items:    make(map[string]users.UserItem),
		settings: make(map[string]users.UserSettings),
		relation: make(map[string]users.Relationship),
	}
}

func (f *fakeUsers) seed(userID, nickname string, settings users.UserSettings) {
	f.items[userID] = users.UserItem{User: users.UserInfo{UserID: userID, Nickname: nickname}}
	f.settings[userID] = settings
}

func (f *fakeUsers) OpenTheDoor(_ context.Context, userID string) (users.UserItem, bool, error) {
	item, ok := f.items[userID]
	return item, ok, nil
}

func (f *fakeUsers) BatchGet(_ context.Context, userIDs []string) (map[string]users.UserInfo, error) {
	out := make(map[string]users.UserInfo, len(userIDs))
	for _, id := range userIDs {
		if item, ok := f.items[id]; ok {
			out[id] = item.User
		}
	}
	return out, nil
}

func (f *fakeUsers) SettingsFor(_ context.Context, userID string) (users.UserSettings, bool, error) {
	s, ok := f.settings[userID]
	return s, ok, nil
}

func (f *fakeUsers) BlockStateBetween(context.Context, string, string) (bool, bool, error) {
	return false, false, nil
}

func (f *fakeUsers) Relationship(_ context.Context, viewerUserID, targetUserID string) (users.Relationship, error) {
	return f.relation[viewerUserID+"|"+targetUserID], nil
}

func (f *fakeUsers) RelationshipBatch(_ context.Context, viewerUserID string, targetUserIDs []string) (map[string]users.Relationship, error) {
	out := make(map[string]users.Relationship, len(targetUserIDs))
	for _, id := range targetUserIDs {
		out[id] = f.relation[viewerUserID+"|"+id]
	}
	return out, nil
}

func (f *fakeUsers) ChatTargetsContext(_ context.Context, _ string, targetUserIDs []string) (map[string]users.ChatTargetContext, error) {
	out := make(map[string]users.ChatTargetContext, len(targetUserIDs))
	for _, id := range targetUserIDs {
		item, ok := f.items[id]
		if !ok {
			continue
		}
		settings := f.settings[id]
		out[id] = users.ChatTargetContext{
			Info:             item.User,
			Relation:         f.relation["|"+id],
			AllowPrivateChat: settings.AllowPrivateChat,
			SettingsFound:    true,
		}
	}
	return out, nil
}
