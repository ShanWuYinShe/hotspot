package reactions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
)

func TestInMemoryAggregateBatchAndReactedTargets(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// 空输入
	batch, err := repo.AggregateBatch(ctx, ReactTargetMoment, nil, "viewer")
	if err != nil || len(batch) != 0 {
		t.Fatalf("empty batch: %#v err=%v", batch, err)
	}

	if _, err := repo.React(ctx, ReactTargetMoment, "m1", "u1", "👍"); err != nil {
		t.Fatalf("react: %v", err)
	}
	if _, err := repo.React(ctx, ReactTargetMoment, "m1", "u2", "👍"); err != nil {
		t.Fatalf("react: %v", err)
	}
	if _, err := repo.React(ctx, ReactTargetMoment, "m1", "u1", "❤️"); err != nil {
		t.Fatalf("react switch emoji: %v", err)
	}

	batch, err = repo.AggregateBatch(ctx, ReactTargetMoment, []string{"m1", "m2"}, "u1")
	if err != nil || len(batch) != 2 {
		t.Fatalf("batch: %#v err=%v", batch, err)
	}
	agg := batch["m1"]
	if len(agg) != 2 {
		t.Fatalf("expected 2 emoji aggregates, got %#v", agg)
	}
	// u1 已切到 ❤️，所以 👍 的 reacted_by_me 应为 false
	for _, r := range agg {
		if r.Emoji == "❤️" && !r.ReactedByMe {
			t.Fatalf("❤️ should be marked reacted_by_me: %#v", agg)
		}
		if r.Emoji == "👍" && r.ReactedByMe {
			t.Fatalf("👍 should not be marked reacted_by_me: %#v", agg)
		}
	}

	ids, next, err := repo.ListReactedTargets(ctx, ReactTargetMoment, "u1", 10, "")
	if err != nil || len(ids) != 1 || ids[0] != "m1" || next != "" {
		t.Fatalf("list reacted targets: %#v next=%q err=%v", ids, next, err)
	}
}

func TestInMemoryUnreactAndListEdges(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// 未反应时 unreact 为 no-op
	deleted, err := repo.Unreact(ctx, ReactTargetMoment, "m1", "u1")
	if err != nil || deleted {
		t.Fatalf("unreact on missing should be no-op: deleted=%v err=%v", deleted, err)
	}

	if _, err := repo.React(ctx, ReactTargetMoment, "m1", "u1", "👍"); err != nil {
		t.Fatalf("react: %v", err)
	}
	deleted, err = repo.Unreact(ctx, ReactTargetMoment, "m1", "u1")
	if err != nil || !deleted {
		t.Fatalf("unreact: deleted=%v err=%v", deleted, err)
	}

	// 空列表
	items, next, err := repo.ListReactions(ctx, ReactTargetMoment, "m1", "", 10, "")
	if err != nil || len(items) != 0 || next != "" {
		t.Fatalf("empty list: %#v next=%q err=%v", items, next, err)
	}
}

type missingTargetProvider struct {
	found bool
}

func (p missingTargetProvider) ReactTarget() ReactTarget { return ReactTargetMoment }

func (p missingTargetProvider) Resolve(context.Context, string) (TargetContext, bool, error) {
	return TargetContext{}, p.found, nil
}

func TestHandlerSetAndListErrorPaths(t *testing.T) {
	// Set: 目标不存在 → httpx.CodeNotFound
	provider := missingTargetProvider{found: false}
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil))
	req := httptest.NewRequest(http.MethodPost, "/set", strings.NewReader(`{"target_type":"moment","target_id":"missing","emoji":"👍","active":true}`))
	rr := httptest.NewRecorder()
	handler.Set(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := decodeReactionResp(rr, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound for missing target, got %d", resp.Code)
	}

	// List: 非法 target_type → httpx.CodeInvalidRequest
	req = httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"target_type":"bogus","target_id":"m1"}`))
	rr = httptest.NewRecorder()
	handler.List(rr, req)
	if err := decodeReactionResp(rr, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest for invalid target type, got %d", resp.Code)
	}

	// List: 空 target_id → httpx.CodeInvalidRequest
	req = httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"target_type":"moment","target_id":""}`))
	rr = httptest.NewRecorder()
	handler.List(rr, req)
	if err := decodeReactionResp(rr, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest for empty target id, got %d", resp.Code)
	}

	// Set: 服务层错误 → httpx.CodeInternalUnavailable
	failing := &reactionErrorRepo{setErr: errors.New("boom")}
	handler = NewHandler(NewServiceWithDeps(failing, []TargetProvider{missingTargetProvider{found: true}}, nil, nil))
	req = httptest.NewRequest(http.MethodPost, "/set", strings.NewReader(`{"target_type":"moment","target_id":"m1","emoji":"👍","active":true}`))
	rr = httptest.NewRecorder()
	handler.Set(rr, req)
	if err := decodeReactionResp(rr, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected httpx.CodeInternalUnavailable on service error, got %d", resp.Code)
	}
}

func decodeReactionResp(rr *httptest.ResponseRecorder, v any) error {
	return json.Unmarshal(rr.Body.Bytes(), v)
}

type reactionErrorRepo struct {
	setErr error
}

func (r *reactionErrorRepo) React(context.Context, ReactTarget, string, string, string) (bool, error) {
	return false, r.setErr
}
func (r *reactionErrorRepo) Unreact(context.Context, ReactTarget, string, string) (bool, error) {
	return false, r.setErr
}
func (r *reactionErrorRepo) ListReactions(context.Context, ReactTarget, string, string, int, string) ([]ReactionItem, string, error) {
	return nil, "", r.setErr
}
func (r *reactionErrorRepo) Aggregate(context.Context, ReactTarget, string, string) ([]Reaction, error) {
	return nil, r.setErr
}
func (r *reactionErrorRepo) AggregateBatch(context.Context, ReactTarget, []string, string) (map[string][]Reaction, error) {
	return nil, r.setErr
}
func (r *reactionErrorRepo) ListReactedTargets(context.Context, ReactTarget, string, int, string) ([]string, string, error) {
	return nil, "", r.setErr
}

func TestHandlerListMissingTargetReturnsEmpty(t *testing.T) {
	// service.List 是纯查询（无目标存在性校验）：不存在的目标返回空列表。
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{missingTargetProvider{found: false}}, nil, nil)
	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"target_type":"moment","target_id":"missing"}`))
	rr := httptest.NewRecorder()
	handler.List(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items []ReactionItem `json:"items"`
		} `json:"data"`
	}
	if err := decodeReactionResp(rr, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 || resp.Data.Items == nil {
		t.Fatalf("expected success with empty items, got %#v", resp)
	}
}

// fixedEmojiConfig 返回仅允许 👍 的 appconfig source。
type fixedEmojiConfig struct{}

func (fixedEmojiConfig) Get(context.Context) appconfig.AppConfig {
	return appconfig.AppConfig{Limits: appconfig.Limits{ReactionEmojis: []string{"👍"}}}
}

func TestEmojiAllowedBranch(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	if !svc.emojiAllowed(context.Background(), "👍") {
		t.Fatal("degraded mode should allow every emoji")
	}
	svc.SetAppConfig(fixedEmojiConfig{})
	if !svc.emojiAllowed(context.Background(), "👍") {
		t.Fatal("👍 must be allowed")
	}
	if svc.emojiAllowed(context.Background(), "💀") {
		t.Fatal("💀 must be rejected")
	}
}

func TestInMemoryListReactionsPagination(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	for _, u := range []string{"u1", "u2", "u3", "u4", "u5"} {
		if _, err := repo.React(ctx, ReactTargetMoment, "m1", u, "👍"); err != nil {
			t.Fatalf("react %s: %v", u, err)
		}
	}
	page1, next, err := repo.ListReactions(ctx, ReactTargetMoment, "m1", "", 2, "")
	if err != nil || len(page1) != 2 || next == "" {
		t.Fatalf("page1: %d items next=%q err=%v", len(page1), next, err)
	}
	page2, next2, err := repo.ListReactions(ctx, ReactTargetMoment, "m1", "", 2, next)
	if err != nil || len(page2) != 2 || next2 == "" {
		t.Fatalf("page2: %d items next=%q err=%v", len(page2), next2, err)
	}
	// 去重：两页不重叠
	seen := map[string]bool{}
	for _, it := range page1 {
		seen[it.User.UserID] = true
	}
	for _, it := range page2 {
		if seen[it.User.UserID] {
			t.Fatalf("overlapping page: %s", it.User.UserID)
		}
	}
	// 最后一页
	page3, next3, err := repo.ListReactions(ctx, ReactTargetMoment, "m1", "", 2, next2)
	if err != nil || len(page3) != 1 || next3 != "" {
		t.Fatalf("page3: %d items next=%q err=%v", len(page3), next3, err)
	}
}

func TestInMemoryListReactedTargetsCursor(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	for _, m := range []string{"m1", "m2", "m3"} {
		if _, err := repo.React(ctx, ReactTargetMoment, m, "u1", "👍"); err != nil {
			t.Fatalf("react %s: %v", m, err)
		}
	}
	ids, next, err := repo.ListReactedTargets(ctx, ReactTargetMoment, "u1", 2, "")
	if err != nil || len(ids) != 2 || next == "" {
		t.Fatalf("first page: %#v next=%q err=%v", ids, next, err)
	}
	ids2, next2, err := repo.ListReactedTargets(ctx, ReactTargetMoment, "u1", 2, next)
	if err != nil || len(ids2) != 1 || next2 != "" {
		t.Fatalf("second page: %#v next=%q err=%v", ids2, next2, err)
	}
	// 游标不匹配时从头开始（分页库常见语义）
	ids3, _, err := repo.ListReactedTargets(ctx, ReactTargetMoment, "u1", 2, "no-such-target")
	if err != nil || len(ids3) != 2 {
		t.Fatalf("unknown cursor should restart from head: %#v err=%v", ids3, err)
	}
}
