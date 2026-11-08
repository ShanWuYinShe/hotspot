package share

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type shareRecordingCounter struct {
	momentIDs []string
	deltas    []int
}

func (c *shareRecordingCounter) AdjustInteractionCount(_ context.Context, momentID string, delta int) {
	c.momentIDs = append(c.momentIDs, momentID)
	c.deltas = append(c.deltas, delta)
}

// shareResolveErrRepo 在 shareStubRepo 基础上让按分享码解析失败（覆盖 Revoke 的
// ResolveShareIDByCode 错误分支）。
type shareResolveErrRepo struct {
	*shareStubRepo
	err error
}

func (r *shareResolveErrRepo) ResolveShareIDByCode(context.Context, string) (string, bool, error) {
	return "", false, r.err
}

func validShareLocationPayload() map[string]any {
	return map[string]any{
		"shared_location": map[string]any{
			"point":         map[string]any{"latitude": 35.1, "longitude": 112.2},
			"location_name": "云龙湖",
		},
	}
}

func TestShareServiceCreateInvalidPrincipalAndExpiry(t *testing.T) {
	svc := NewServiceWithRepo(&shareStubRepo{})

	// 未登录（无 principal）→ 参数错误
	if _, err := svc.Create(context.Background(), CreateShareRequest{ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("create without principal err = %v, want ErrInvalidShareRequest", err)
	}
	// 负过期时间 → 参数错误（ExpiresInSeconds <= 0 分支）
	if _, err := svc.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeLocation, Payload: validShareLocationPayload(), ExpiresInSeconds: -1}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("create with negative expiry err = %v, want ErrInvalidShareRequest", err)
	}
	// 合法正数过期时间 → 透传 expiresAt 到仓储
	repo := &shareStubRepo{}
	svc2 := NewServiceWithRepo(repo)
	if _, err := svc2.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeLocation, Payload: validShareLocationPayload(), ExpiresInSeconds: 60}); err != nil {
		t.Fatalf("create with expiry: %v", err)
	}
	if repo.createCmd.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be computed")
	}
}

// TestShareServiceCreateAdjustsMomentInteractionCount 覆盖 InteractionCounter 注入：
// 仅 moment 类型分享 +1，其它类型不调整（且 SetInteractionCounter 覆盖）。
func TestShareServiceCreateAdjustsMomentInteractionCount(t *testing.T) {
	repo := &shareStubRepo{}
	counter := &shareRecordingCounter{}
	svc := NewServiceWithRepo(repo)
	svc.SetInteractionCounter(counter)

	if _, err := svc.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": "moment_42"}}); err != nil {
		t.Fatalf("create moment share: %v", err)
	}
	if len(counter.momentIDs) != 1 || counter.momentIDs[0] != "moment_42" || counter.deltas[0] != 1 {
		t.Fatalf("unexpected interaction adjustments: %#v %#v", counter.momentIDs, counter.deltas)
	}

	if _, err := svc.Create(shareContextWithUser("u1"), CreateShareRequest{ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}}); err != nil {
		t.Fatalf("create user share: %v", err)
	}
	if len(counter.momentIDs) != 1 {
		t.Fatalf("non-moment share must not adjust interaction count: %#v", counter.momentIDs)
	}
}

// TestShareServiceValidateShareCode 覆盖 ValidateShareCode（spec 消息发送前校验）：
// 非法码、仓储错误透传、有效码通过。
func TestShareServiceValidateShareCode(t *testing.T) {
	invalidSvc := NewServiceWithRepo(&shareStubRepo{})
	if err := invalidSvc.ValidateShareCode(context.Background(), "bad"); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("invalid code err = %v, want ErrInvalidShareRequest", err)
	}

	boom := errors.New("validate boom")
	repoErrSvc := NewServiceWithRepo(&shareErrorRepo{getErr: boom})
	if err := repoErrSvc.ValidateShareCode(context.Background(), "ABCDEFGHJKLM"); !errors.Is(err, boom) {
		t.Fatalf("repo error must propagate, got %v", err)
	}

	svc := NewService()
	created, err := svc.Create(shareContextWithUser("user_alice"), CreateShareRequest{ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ValidateShareCode(context.Background(), strings.ToLower(created.ShareCode)); err != nil {
		t.Fatalf("valid code must pass: %v", err)
	}
}

func TestShareServiceRevokeInvalidPrincipalAndResolveFailure(t *testing.T) {
	svc := NewServiceWithRepo(&shareStubRepo{})
	if err := svc.Revoke(context.Background(), RevokeShareRequest{ShareID: "share_1"}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("revoke without principal err = %v, want ErrInvalidShareRequest", err)
	}

	boom := errors.New("resolve boom")
	repo := &shareResolveErrRepo{shareStubRepo: &shareStubRepo{}, err: boom}
	if err := NewServiceWithRepo(repo).Revoke(shareContextWithUser("u1"), RevokeShareRequest{ShareCode: "ABCDEFGHJKLM"}); !errors.Is(err, boom) {
		t.Fatalf("resolve error must propagate, got %v", err)
	}
}

func TestShareServiceListValidation(t *testing.T) {
	svc := NewServiceWithRepo(&shareStubRepo{})
	if _, err := svc.List(context.Background(), ListSharesRequest{}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("list without principal err = %v, want ErrInvalidShareRequest", err)
	}
	if _, err := svc.List(shareContextWithUser("u1"), ListSharesRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("invalid cursor err = %v, want ErrInvalidShareRequest", err)
	}
}

func TestNewListSharesQueryValidation(t *testing.T) {
	if _, err := newListSharesQuery("", ListSharesRequest{}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("empty creator err = %v, want ErrInvalidShareRequest", err)
	}
	if _, err := newListSharesQuery("u1", ListSharesRequest{ShareType: "bogus"}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("bogus share type err = %v, want ErrInvalidShareRequest", err)
	}
	// 空白 share_type 视为不筛选
	q, err := newListSharesQuery("u1", ListSharesRequest{ShareType: "   "})
	if err != nil || q.ShareType != "" {
		t.Fatalf("blank share type must be ignored: %#v err=%v", q, err)
	}
	// limit 上限裁剪（normalizeSharePagination 的 max 分支）
	q, err = newListSharesQuery("u1", ListSharesRequest{Pagination: &PaginationParams{Limit: maxShareListLimit + 50}})
	if err != nil || q.Limit != maxShareListLimit {
		t.Fatalf("limit must be clamped to %d: %#v err=%v", maxShareListLimit, q, err)
	}
	// 可解码但字段缺失的游标 → 拒绝
	forged, encErr := shareCursorCodec.Encode(shareCursor{})
	if encErr != nil {
		t.Fatalf("encode forged cursor: %v", encErr)
	}
	if _, err := newListSharesQuery("u1", ListSharesRequest{Pagination: &PaginationParams{Cursor: forged}}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("forged cursor err = %v, want ErrInvalidShareRequest", err)
	}
}

// TestNormalizeSharePayloadAllTypes 逐类型覆盖入参规范化与全部拒绝分支
// （字段类型/空白/多余字段/坐标范围/名称长度）。
func TestNormalizeSharePayloadAllTypes(t *testing.T) {
	locationPayload := validShareLocationPayload()
	locationInner := locationPayload["shared_location"].(map[string]any)

	cases := []struct {
		name      string
		shareType string
		payload   map[string]any
		wantNil   bool
	}{
		{"user ok", ShareTypeUser, map[string]any{"shared_user_id": " u2 "}, false},
		{"user non-string", ShareTypeUser, map[string]any{"shared_user_id": 2}, true},
		{"user blank", ShareTypeUser, map[string]any{"shared_user_id": "  "}, true},
		{"user extra field", ShareTypeUser, map[string]any{"shared_user_id": "u2", "x": "y"}, true},
		{"group ok", ShareTypeGroup, map[string]any{"shared_group_id": " g1 "}, false},
		{"group missing", ShareTypeGroup, map[string]any{}, true},
		{"group extra field", ShareTypeGroup, map[string]any{"shared_group_id": "g1", "x": 1.0}, true},
		{"moment ok", ShareTypeMoment, map[string]any{"shared_moment_id": "m1"}, false},
		{"moment blank", ShareTypeMoment, map[string]any{"shared_moment_id": " "}, true},
		{"location ok", ShareTypeLocation, locationPayload, false},
		{"location no payload key", ShareTypeLocation, map[string]any{}, true},
		{"location wrong container", ShareTypeLocation, map[string]any{"shared_location": "x"}, true},
		{"location extra top field", ShareTypeLocation, map[string]any{"shared_location": locationInner, "x": 1.0}, true},
		{"location missing point", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"location_name": "x"}}, true},
		{"location point wrong type", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": "x", "location_name": "x"}}, true},
		{"location lat non-float", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35, "longitude": 112.0}, "location_name": "x"}}, true},
		{"location lng non-float", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 112}, "location_name": "x"}}, true},
		{"location name non-string", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 112.0}, "location_name": 1}}, true},
		{"location lat out of range", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 91.0, "longitude": 112.0}, "location_name": "x"}}, true},
		{"location lng out of range", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 181.0}, "location_name": "x"}}, true},
		{"location blank name", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 112.0}, "location_name": "   "}}, true},
		{"location name too long", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 112.0}, "location_name": strings.Repeat("a", 101)}}, true},
		{"location extra location field", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 112.0}, "location_name": "x", "extra": 1.0}}, true},
		{"location extra point field", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"point": map[string]any{"latitude": 35.0, "longitude": 112.0, "altitude": 3.0}, "location_name": "x"}}, true},
		{"unknown type", "bogus", locationPayload, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeSharePayload(tc.shareType, tc.payload)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("payload = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("payload = nil, want normalized payload")
			}
		})
	}

	// 规范化结果：空白 trim + 结构收敛
	userPayload := normalizeSharePayload(ShareTypeUser, map[string]any{"shared_user_id": " u2 "})
	if userPayload["shared_user_id"] != "u2" || len(userPayload) != 1 {
		t.Fatalf("user payload = %#v", userPayload)
	}
	loc := normalizeSharePayload(ShareTypeLocation, locationPayload)
	shared, ok := loc["shared_location"].(map[string]any)
	if !ok || len(shared) != 2 || shared["location_name"] != "云龙湖" {
		t.Fatalf("location payload = %#v", loc)
	}
	point, ok := shared["point"].(map[string]any)
	if !ok || len(point) != 2 || point["latitude"] != 35.1 || point["longitude"] != 112.2 {
		t.Fatalf("location point = %#v", point)
	}
}
