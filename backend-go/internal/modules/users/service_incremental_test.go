package users

import (
	"context"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func incrementalTestCtx() context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_p1", SessionID: "sess_p1"})
}

// 审查 P1-3 防回归：update_account 只写请求携带的列——只改 bio 时，
// nickname/avatar/tags 等其余列不得被旧快照回滚。
func TestUpdateMyProfileOnlyTouchesPatchedColumns(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := incrementalTestCtx()

	if err := repo.PutProfile(ctx, MyProfile{
		UserID: "user_p1", Nickname: "原昵称", AvatarURL: "https://cdn/a.png",
		Bio: "旧简介", Tags: []string{"tag1", "tag2"},
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	bio := "新简介"
	if err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{Bio: &bio}); err != nil {
		t.Fatalf("update bio: %v", err)
	}

	profile, ok, err := repo.GetProfile(ctx, "user_p1")
	if err != nil || !ok {
		t.Fatalf("get profile: ok=%v err=%v", ok, err)
	}
	if profile.Bio != "新简介" {
		t.Fatalf("patched bio must persist, got %q", profile.Bio)
	}
	if profile.Nickname != "原昵称" || profile.AvatarURL != "https://cdn/a.png" || len(profile.Tags) != 2 {
		t.Fatalf("unpatched columns must stay: %#v", profile)
	}
}

// 审查 P1-3 防回归：update_location 与 update_account 并发交错不再互相
// 覆盖——两者各自只写自己的列（旧全量回写路径下后写者会回滚前者）。
func TestUpdateLocationAndProfileDoNotClobberEachOther(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := incrementalTestCtx()

	if err := repo.PutProfile(ctx, MyProfile{UserID: "user_p1", Nickname: "原昵称"}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	nickname := "新昵称"
	if err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{Nickname: &nickname}); err != nil {
		t.Fatalf("update nickname: %v", err)
	}
	if err := svc.UpdateLocation(ctx, UpdateLocationRequest{Location: Location{
		LocationName: "上海",
		Point:        GeoPoint{Latitude: 31.23, Longitude: 121.47},
	}}); err != nil {
		t.Fatalf("update location: %v", err)
	}

	profile, ok, err := repo.GetProfile(ctx, "user_p1")
	if err != nil || !ok {
		t.Fatalf("get profile: ok=%v err=%v", ok, err)
	}
	if profile.Nickname != "新昵称" {
		t.Fatalf("location update must not roll back nickname, got %q", profile.Nickname)
	}
	if profile.Location.LocationName != "上海" {
		t.Fatalf("location must persist, got %#v", profile.Location)
	}
}

// 审查 P1-3 防回归：PutProfile（建档/Upgrade 路径）不得覆写运行时
// presence 状态——与 PG 版 ON CONFLICT 不更新 presence 列同口径。
func TestPutProfilePreservesPresenceOnOverwrite(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := incrementalTestCtx()

	if err := repo.PutProfile(ctx, MyProfile{UserID: "user_p1", PresenceStatus: UserPresenceOnline}); err != nil {
		t.Fatalf("seed online: %v", err)
	}
	// 调用方快照读取于上线之前（offline），回写不得回滚 online。
	if err := repo.PutProfile(ctx, MyProfile{UserID: "user_p1", PresenceStatus: UserPresenceOffline}); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	profile, _, _ := repo.GetProfile(ctx, "user_p1")
	if profile.PresenceStatus != UserPresenceOnline {
		t.Fatalf("presence must survive overwrite, got %q", profile.PresenceStatus)
	}
}

// 审查 P1-3 防回归：update_settings 只写 patch 携带的列；设置行缺失时
// 按默认值建档并套用 patch。
func TestUpdateMySettingsPatchesOnlyProvidedFields(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := incrementalTestCtx()

	// 行缺失：建档路径（默认值 + patch）。
	searchable := false
	if err := svc.UpdateMySettings(ctx, UpdateSettingsRequest{Settings: PartialUserSettings{Searchable: &searchable}}); err != nil {
		t.Fatalf("create via patch: %v", err)
	}
	settings, ok, err := repo.GetSettings(ctx, "user_p1")
	if err != nil || !ok {
		t.Fatalf("get settings: ok=%v err=%v", ok, err)
	}
	if settings.Searchable || !settings.NicknameSearchable || settings.AllowPrivateChat != ChatPermEveryone {
		t.Fatalf("expect defaults with patch applied: %#v", settings)
	}

	// 已有行：只改 searchable，其余列不得回滚。
	nicknameSearchable := false
	if err := svc.UpdateMySettings(ctx, UpdateSettingsRequest{Settings: PartialUserSettings{NicknameSearchable: &nicknameSearchable}}); err != nil {
		t.Fatalf("patch existing: %v", err)
	}
	settings, _, _ = repo.GetSettings(ctx, "user_p1")
	if settings.NicknameSearchable {
		t.Fatal("patched nickname_searchable must persist")
	}
	if !settings.LocationVisible || !settings.StatsVisible {
		t.Fatalf("unpatched settings must stay: %#v", settings)
	}
}

// 审查 P1-3 防回归：update_account 命中缺失行时走默认建档路径（行为与
// 旧实现一致），patch 字段与标签一并落上。
func TestUpdateMyProfileCreatesDefaultRowWhenMissing(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := incrementalTestCtx()

	nickname := "首个昵称"
	tags := []string{"hello"}
	if err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{Nickname: &nickname, Tags: &tags}); err != nil {
		t.Fatalf("create via patch: %v", err)
	}
	profile, ok, err := repo.GetProfile(ctx, "user_p1")
	if err != nil || !ok {
		t.Fatalf("get profile: ok=%v err=%v", ok, err)
	}
	if profile.Nickname != "首个昵称" || len(profile.Tags) != 1 || profile.Tags[0] != "hello" {
		t.Fatalf("default row must carry patch: %#v", profile)
	}
}
