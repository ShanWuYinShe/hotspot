package share

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type shareStubRepo struct {
	createCmd createShareCommand
	getQuery  getShareQuery
	listQuery listSharesQuery
	revokeCmd revokeShareCommand
	err       error
}

func (r *shareStubRepo) Create(_ context.Context, cmd createShareCommand) (ShareCreated, error) {
	r.createCmd = cmd
	return ShareCreated{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM"}, r.err
}

// ResolveShareIDByCode 撤销幂等路径的桩：默认把 share_code 映射为同值
// share_id（Get 桩被置为 not found 时仍可定位到分享，模拟「已 revoked」）。
func (r *shareStubRepo) ResolveShareIDByCode(_ context.Context, shareCode string) (string, bool, error) {
	if shareCode == "" {
		return "", false, nil
	}
	return "share_" + shareCode, true, nil
}

func (r *shareStubRepo) Get(_ context.Context, query getShareQuery) (ShareDetail, error) {
	r.getQuery = query
	return ShareDetail{}, r.err
}
func (r *shareStubRepo) List(_ context.Context, query listSharesQuery) (PaginatedShares, error) {
	r.listQuery = query
	return PaginatedShares{}, r.err
}
func (r *shareStubRepo) Revoke(_ context.Context, cmd revokeShareCommand) error {
	r.revokeCmd = cmd
	return r.err
}

func shareContextWithUser(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func TestServiceCreateValidatesRequest(t *testing.T) {
	service := NewServiceWithRepo(&shareStubRepo{})
	if _, err := service.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: "bad", Payload: map[string]any{}}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("expected invalid share request, got %v", err)
	}
	if _, err := service.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeLocation, Payload: map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 100.0, "longitude": 1.0}, "location_name": "x"}}}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("expected invalid location payload, got %v", err)
	}
	if _, err := service.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}, ExpiresInSeconds: maxShareExpiresInSec + 1}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("expected invalid expires, got %v", err)
	}
}

func TestServiceCreatePassesNormalizedCommand(t *testing.T) {
	repo := &shareStubRepo{}
	service := NewServiceWithRepo(repo)
	_, err := service.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if repo.createCmd.CreatorUserID != "u1" || repo.createCmd.ShareType != ShareTypeUser || repo.createCmd.Payload["shared_user_id"] != "u2" {
		t.Fatalf("unexpected create command: %#v", repo.createCmd)
	}
}

func TestServiceGetValidatesShareCode(t *testing.T) {
	service := NewServiceWithRepo(&shareStubRepo{})
	if _, err := service.Get(context.Background(), GetShareRequest{ShareCode: "bad"}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("expected invalid share code, got %v", err)
	}
}

func TestServiceListNormalizesPagination(t *testing.T) {
	repo := &shareStubRepo{}
	service := NewServiceWithRepo(repo)
	_, err := service.List(shareContextWithUser("u1"), ListSharesRequest{ShareType: ShareTypeUser, Pagination: &PaginationParams{Limit: 0}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.listQuery.CreatorUserID != "u1" || repo.listQuery.ShareType != ShareTypeUser || repo.listQuery.Limit != defaultShareListLimit {
		t.Fatalf("unexpected list query: %#v", repo.listQuery)
	}
}

// TestServiceRevokeByCodeIsIdempotentForRevokedShare 防回归（审查 P2-8）：
// 按 share_code 撤销已撤销/已过期的分享必须幂等成功。此前走 repo.Get，而
// Get 把 revoked 统一映射为 ErrShareNotFound → 返回 404，与按 share_id 的
// 幂等口径分裂。桩里 Get 恒返回 not found，ResolveShareIDByCode 仍能定位，
// 断言请求到达 Revoke 且无错误。
func TestServiceRevokeByCodeIsIdempotentForRevokedShare(t *testing.T) {
	repo := &shareStubRepo{}
	service := NewServiceWithRepo(repo)
	if err := service.Revoke(shareContextWithUser("u1"), RevokeShareRequest{ShareCode: "abcdefgh"}); err != nil {
		t.Fatalf("revoke by code of revoked share must be idempotent, got %v", err)
	}
	if repo.revokeCmd.ShareID != "share_abcdefgh" {
		t.Fatalf("unexpected revoke command: %#v", repo.revokeCmd)
	}
}

func TestServiceRevokeValidatesRequest(t *testing.T) {
	service := NewServiceWithRepo(&shareStubRepo{})
	if err := service.Revoke(shareContextWithUser("u1"), RevokeShareRequest{}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("expected invalid revoke request, got %v", err)
	}
}
