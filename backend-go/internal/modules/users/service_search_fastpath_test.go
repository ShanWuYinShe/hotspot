package users

import (
	"context"
	"errors"
	"testing"
)

// fastPathRepo 模拟 Postgres 搜索返回的 Settings 快路径：SearchProfiles 自带
// 可见性，Service 不应再调用 GetSettingsBatch。
type fastPathRepo struct {
	Repository
	settingsCalls int
}

func (r *fastPathRepo) SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error) {
	return PaginatedProfiles{
		Items: []MyProfile{{UserID: "user_bob", Nickname: "Bob"}},
		Settings: map[string]UserSettings{
			"user_bob": {PresenceVisible: true, LocationVisible: true, StatsVisible: true},
		},
	}, nil
}

func (r *fastPathRepo) GetSettingsBatch(context.Context, []string) (map[string]UserSettings, error) {
	r.settingsCalls++
	return nil, errors.New("GetSettingsBatch must not be called when search inlines settings")
}

func TestSearchUsesInlineSettingsWithoutBatchQuery(t *testing.T) {
	repo := &fastPathRepo{}
	service := NewServiceWithRepo(repo)
	result, err := service.Search(context.Background(), SearchUsersRequest{Keyword: "Bob"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if repo.settingsCalls != 0 {
		t.Fatalf("GetSettingsBatch calls = %d, want 0 (inline settings fast path)", repo.settingsCalls)
	}
	if len(result.Items) != 1 || result.Items[0].User.UserID != "user_bob" {
		t.Fatalf("unexpected items: %#v", result.Items)
	}
}
