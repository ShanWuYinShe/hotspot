package users

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func p2TestCtx() context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_p2", SessionID: "sess_p2"})
}

// 审查 P2-19 防回归：privacy_level 仅接受 public/private（spec
// api_spec_types.md PrivacyLevel）。非法值一旦落库是 fail-closed——搜索
// 谓词 privacy_level='public' 不命中，用户从搜索结果中静默消失。
func TestUpdateMySettingsRejectsInvalidPrivacyLevel(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := p2TestCtx()

	bad := "friends"
	if err := svc.UpdateMySettings(ctx, UpdateSettingsRequest{Settings: PartialUserSettings{PrivacyLevel: &bad}}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest for privacy_level=friends, got %v", err)
	}
	settings, ok, _ := repo.GetSettings(ctx, "user_p2")
	if ok && settings.PrivacyLevel == "friends" {
		t.Fatal("invalid privacy_level must not be persisted")
	}

	// 合法枚举照常接受。
	private := "private"
	if err := svc.UpdateMySettings(ctx, UpdateSettingsRequest{Settings: PartialUserSettings{PrivacyLevel: &private}}); err != nil {
		t.Fatalf("expected private accepted, got %v", err)
	}
	settings, _, _ = repo.GetSettings(ctx, "user_p2")
	if settings.PrivacyLevel != "private" {
		t.Fatalf("expected privacy_level=private persisted, got %q", settings.PrivacyLevel)
	}
}

// 审查 P2-14 防回归：内存版 SearchProfiles 对齐 PG SQL 谓词——过滤
// is_deleted=FALSE 与 privacy_level='public'。此前两个过滤都缺失：注销
// 用户（settings 已删走默认 searchable=true）与 private 用户都能被搜出。
func TestMemorySearchProfilesFiltersDeletedAndNonPublic(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()

	seed := func(id, nickname, privacyLevel string, deleted bool) {
		if err := repo.PutProfile(ctx, MyProfile{UserID: id, Nickname: nickname, IsDeleted: deleted}); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
		if deleted {
			return // 注销路径会删除 settings（模拟 DeleteProfile 后状态）
		}
		settings := defaultSettings()
		settings.PrivacyLevel = privacyLevel
		if err := repo.PutSettings(ctx, id, settings); err != nil {
			t.Fatalf("seed settings %s: %v", id, err)
		}
	}
	seed("user_pub", "公开用户", "public", false)
	seed("user_priv", "私域用户", "private", false)
	seed("user_del", "注销用户", "", true)

	result, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "用户"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	found := map[string]bool{}
	for _, profile := range result.Items {
		found[profile.UserID] = true
	}
	if !found["user_pub"] {
		t.Fatal("public searchable user must be found")
	}
	if found["user_priv"] {
		t.Fatal("non-public privacy_level user must be excluded (PG parity)")
	}
	if found["user_del"] {
		t.Fatal("deleted user must be excluded (PG parity)")
	}
}
