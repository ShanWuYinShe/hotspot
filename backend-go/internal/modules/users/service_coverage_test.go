package users

import (
	"context"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestResolveReferencePointChain(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)

	// 1. 显式传入优先
	ref := GeoPoint{Latitude: 10, Longitude: 20}
	got := svc.resolveReferencePoint(context.Background(), ref)
	if got != ref {
		t.Fatalf("explicit ref must win, got %#v", got)
	}

	// 2. 已登录用户最近位置（无显式 ref）
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1", SessionID: "sess_1"})
	_ = repo.PutProfile(ctx, MyProfile{UserID: "u1", Location: Location{Point: GeoPoint{Latitude: 31.2, Longitude: 121.5}}})
	got = svc.resolveReferencePoint(ctx, GeoPoint{})
	if got.Latitude != 31.2 || got.Longitude != 121.5 {
		t.Fatalf("profile location should be used, got %#v", got)
	}

	// 3. 匿名 + 无 appConfig → 默认回退
	got = svc.resolveReferencePoint(context.Background(), GeoPoint{})
	if got.Latitude == 0 && got.Longitude == 0 {
		t.Fatalf("default fallback expected, got %#v", got)
	}
}

func TestServiceUnfollowErrorPaths(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1", SessionID: "sess_1"})

	// 取关自己 → 冲突
	if err := svc.Unfollow(ctx, "u1"); err != ErrConflict {
		t.Fatalf("self unfollow: expected conflict, got %v", err)
	}
	// 取关不存在的人 → 用户不存在
	if err := svc.Unfollow(ctx, "missing"); err != ErrUserNotFound {
		t.Fatalf("unfollow missing: expected not found, got %v", err)
	}
	// 未登录 → 未授权
	if err := svc.Unfollow(context.Background(), "u2"); err != ErrUnauthorized {
		t.Fatalf("anonymous unfollow: expected unauthorized, got %v", err)
	}
}
