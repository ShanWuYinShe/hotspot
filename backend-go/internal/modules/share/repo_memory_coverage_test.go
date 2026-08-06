package share

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestMemoryShareTargetExistenceBranches(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()

	// 目标不存在的类型 → create 失败
	_, err := repo.Create(ctx, createShareCommand{
		CreatorUserID: "u1",
		ShareType:     ShareTypeUser,
		Payload:       map[string]any{},
	})
	if !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("empty user payload must be not-found, got %v", err)
	}
	// 未知 share type → invalid
	_, err = repo.Create(ctx, createShareCommand{
		CreatorUserID: "u1",
		ShareType:     "bogus",
		Payload:       map[string]any{"x": "y"},
	})
	if !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("bogus type must be invalid, got %v", err)
	}
	// location 类型不需要目标存在
	created, err := repo.Create(ctx, createShareCommand{
		CreatorUserID: "u1",
		ShareType:     ShareTypeLocation,
		Payload:       map[string]any{"lat": 35.0},
	})
	if err != nil || created.ShareID == "" {
		t.Fatalf("location share create: %#v err=%v", created, err)
	}
}

func TestMemorySharePreviewBranches(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()

	// group/moment 类型 preview
	for _, tt := range []struct {
		shareType string
		payload   map[string]any
	}{
		{ShareTypeGroup, map[string]any{"shared_group_id": "g1"}},
		{ShareTypeMoment, map[string]any{"shared_moment_id": "m1"}},
	} {
		created, err := repo.Create(ctx, createShareCommand{CreatorUserID: "u1", ShareType: tt.shareType, Payload: tt.payload})
		if err != nil {
			t.Fatalf("%s create: %v", tt.shareType, err)
		}
		detail, err := repo.Get(ctx, getShareQuery{ShareCode: created.ShareCode})
		if err != nil || detail.Preview == nil {
			t.Fatalf("%s preview: %#v err=%v", tt.shareType, detail, err)
		}
	}

	// moment 类型：payload 缺 id → not found
	if _, err := repo.Create(ctx, createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeMoment, Payload: map[string]any{}}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("missing moment id must be not-found, got %v", err)
	}
}

func TestRevokeByShareCodeAndInvalidRequests(t *testing.T) {
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1", SessionID: "sess_1"})
	repo := NewInMemoryRepository()
	created, err := repo.Create(ctx, createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeLocation, Payload: map[string]any{"lat": 1.0}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	svc := NewServiceWithRepo(repo)

	// 同时传 share_id 与 share_code → 参数错误
	if err := svc.Revoke(ctx, RevokeShareRequest{ShareID: created.ShareID, ShareCode: created.ShareCode}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("both fields must be rejected, got %v", err)
	}
	// 都不传 → 参数错误
	if err := svc.Revoke(ctx, RevokeShareRequest{}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("empty revoke must be rejected, got %v", err)
	}
	// 仅 share_code → 解析并撤销
	if err := svc.Revoke(ctx, RevokeShareRequest{ShareCode: created.ShareCode}); err != nil {
		t.Fatalf("revoke by code: %v", err)
	}
	// 已撤销再撤销 → 成功（幂等）
	if err := svc.Revoke(ctx, RevokeShareRequest{ShareID: created.ShareID}); err != nil {
		t.Fatalf("revoke again should be idempotent, got %v", err)
	}
	// 未知 code → not found
	if err := svc.Revoke(ctx, RevokeShareRequest{ShareCode: "NOPE"}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("unknown code must be not-found, got %v", err)
	}
}
