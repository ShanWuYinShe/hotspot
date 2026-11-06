package users

import (
	"context"
	"errors"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// TestUsersModuleWiringCoverage 覆盖一等模块装配/路由挂载。
func TestUsersModuleWiringCoverage(t *testing.T) {
	factory := persistence.NewFactory(config.Config{})
	source := appconfig.NewStaticSource(config.Load())
	module := NewModule(factory, source, nil)
	router := chi.NewRouter()
	module.Mount(router)
	publicRouter := chi.NewRouter()
	module.MountPublic(publicRouter)
}

// TestUsersServiceWrapperCoverage 覆盖跨模块最小接口的转发方法/依赖注入
// （内存仓储迁移后这些包装方法不再被内存实现用例间接覆盖）。
func TestUsersServiceWrapperCoverage(t *testing.T) {
	ctx := context.Background()
	svc := NewServiceWithDeps(NewPostgresRepository(persistence.NewFactory(config.Config{})), nil, nil)
	if err := svc.SetAccountBanState(ctx, "u1", AccountBanState{}); err == nil {
		t.Fatal("expected non-queryable repo error")
	}
	svc.SetSettingsCache(nil)
	if _, err := svc.Relationship(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected non-queryable repo error")
	}
	if _, err := svc.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected non-queryable repo error")
	}
	if _, err := svc.GetActorInfo(ctx, "u1"); err == nil {
		t.Fatal("expected non-queryable repo error")
	}
	if _, _, err := svc.ChatSettingsFor(ctx, "u1"); err == nil {
		t.Fatal("expected non-queryable repo error")
	}
	if _, err := svc.NearbyMomentNotifyTargets(ctx, 1, 2, 3, "u1", 10); err == nil {
		t.Fatal("expected non-queryable repo error")
	}
	if _, err := svc.ChatTargetsContext(ctx, "viewer", []string{"u1"}); err == nil {
		t.Fatal("expected non-queryable repo error")
	}

	svc.SetContentProviders(&stubGroupsProvider{}, &stubMomentsProvider{})
	if _, err := svc.ListGroupsByUser(ctx, "u1", 10, ""); err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if _, err := svc.CountCommonGroups(ctx, "u1", "u2"); err != nil {
		t.Fatalf("count common groups: %v", err)
	}
	if _, err := svc.ListMomentsByAuthor(ctx, "u1", 10, "", false); err != nil {
		t.Fatalf("list moments: %v", err)
	}
	if _, err := svc.ListReactedByUser(ctx, "u1", "u2", 10, ""); err != nil {
		t.Fatalf("list reacted: %v", err)
	}

	if got := GetActorInfo(ctx, nil, "u1"); got.UserID != "u1" {
		t.Fatalf("actor fallback = %#v", got)
	}
}

type errorRow struct{}

func (errorRow) Scan(...any) error { return errors.New("scan failed") }

// TestPostgresRemainingPathsCoverage 覆盖 PG 仓储在「无数据库后端」下的骨架
// 分支（真实 SQL 语义由 IT 覆盖），把内存仓储迁出后丢失的覆盖率补回。
func TestPostgresRemainingPathsCoverage(t *testing.T) {
	ctx := context.Background()
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	name := "nick"
	if err := repo.SetAccountBanState(ctx, "u1", AccountBanState{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{Nickname: &name}, nil); err == nil {
		t.Fatal("expected error")
	}
	if _, err := repo.UpdateProfileLocation(ctx, "u1", Location{}); err == nil {
		t.Fatal("expected error")
	}
	// 空 patch 在进入可查询性检查前就返回 (false,nil)，仅需覆盖该路径。
	_, _ = repo.UpdateSettings(ctx, "u1", PartialUserSettings{})
	if err := repo.BlockCascade(ctx, "u1", "u2", "spam"); err == nil {
		t.Fatal("expected error")
	}
	if err := repo.SetPresenceStatus(ctx, "u1", UserPresenceOnline); err == nil {
		t.Fatal("expected error")
	}
	if _, err := repo.CheckBlockedTags(ctx, []string{"blocked"}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := repo.NearbyMomentNotifyTargets(ctx, 34.2, 117.1, 5000, "u1", 10); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := repo.GetRecentLocation(ctx, "u1"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := repo.GetProfiles(ctx, []string{"u1"}); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := repo.MutualFriends(ctx, "u1", "u2", 3); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := repo.BlockedEitherWay(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected error")
	}

	if sql, args := buildUpdateProfileSQL("u1", ProfilePatch{Nickname: &name}); sql == "" || len(args) != 2 {
		t.Fatalf("profile update SQL = %q args=%v", sql, args)
	}
	if _, _, ok := buildUpdateSettingsSQL("u1", PartialUserSettings{}); ok {
		t.Fatal("empty settings patch must return ok=false")
	}
	if _, err := scanUserProfileRow(errorRow{}); err == nil {
		t.Fatal("expected scan error")
	}
	_ = types.StrPtr("")
}
