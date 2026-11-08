package reactions

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

const g5ListBody = "{\"target_type\":\"moment\",\"target_id\":\"m1\"}"

// g5ReactionsRepo 是 Repository 的可注入错误测试替身：List/Unreact 返回预设
// 错误，其余读写返回零值。
type g5ReactionsRepo struct {
	listErr    error
	unreactErr error
}

func (r *g5ReactionsRepo) React(context.Context, ReactTarget, string, string, string) (ReactionOutcome, error) {
	return ReactionUnchanged, nil
}

func (r *g5ReactionsRepo) Unreact(context.Context, ReactTarget, string, string) (bool, error) {
	return false, r.unreactErr
}

func (r *g5ReactionsRepo) ListReactions(context.Context, ReactTarget, string, string, int, string) ([]ReactionItem, string, error) {
	return nil, "", r.listErr
}

func (r *g5ReactionsRepo) Aggregate(context.Context, ReactTarget, string, string) ([]Reaction, error) {
	return nil, nil
}

func (r *g5ReactionsRepo) AggregateBatch(context.Context, ReactTarget, []string, string) (map[string][]Reaction, error) {
	return nil, nil
}

func (r *g5ReactionsRepo) ListReactedTargets(context.Context, ReactTarget, string, int, string) ([]string, string, error) {
	return nil, "", nil
}

// g5ErrTargetProvider 是 Resolve 必定报错的 TargetProvider（覆盖服务层把
// provider 错误上抛并记日志的分支）。
type g5ErrTargetProvider struct{ kind ReactTarget }

func (p g5ErrTargetProvider) ReactTarget() ReactTarget { return p.kind }

func (p g5ErrTargetProvider) Resolve(context.Context, string) (TargetContext, bool, error) {
	return TargetContext{}, false, errors.New("provider down")
}

// TestModuleMountsReactionRoutes 覆盖 NewModule 装配与 Mount/MountPublic：
// 公开节点匿名可访问 fetch（非法请求体走 10000），/api/v1 节点必须有鉴权。
func TestModuleMountsReactionRoutes(t *testing.T) {
	module := NewModule(nil, nil, nil, nil)
	if module == nil || module.handler == nil || module.Service == nil {
		t.Fatalf("expected wired module, got %#v", module)
	}

	apiRouter := chi.NewRouter()
	module.Mount(apiRouter)
	publicRouter := chi.NewRouter()
	module.MountPublic(publicRouter)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reactions/fetch", strings.NewReader("{"))
	publicRouter.ServeHTTP(rr, req)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeInvalidRequest {
		t.Fatalf("public fetch invalid body: code = %d, want %d", code, httpx.CodeInvalidRequest)
	}

	// /api/v1 组必须挂鉴权：单测未装配 auth service 时会以 10600 拒绝请求，
	// 但绝不会以成功码 0 放行。
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/reactions/fetch", strings.NewReader(g5ListBody))
	apiRouter.ServeHTTP(rr, req)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeUnauthorized && code != httpx.CodeInternalUnavailable {
		t.Fatalf("unauthenticated api fetch: code = %d, want auth rejection", code)
	}
}

// TestNormalizeReactionLimitClamps 覆盖分页 limit 的默认值与上限钳制。
func TestNormalizeReactionLimitClamps(t *testing.T) {
	if got := normalizeReactionLimit(0); got != 20 {
		t.Fatalf("normalizeReactionLimit(0) = %d, want 20", got)
	}
	if got := normalizeReactionLimit(-5); got != 20 {
		t.Fatalf("normalizeReactionLimit(-5) = %d, want 20", got)
	}
	if got := normalizeReactionLimit(1000); got != 100 {
		t.Fatalf("normalizeReactionLimit(1000) = %d, want 100", got)
	}
	if got := normalizeReactionLimit(30); got != 30 {
		t.Fatalf("normalizeReactionLimit(30) = %d, want 30", got)
	}
}

// TestPostgresRepositoryWithoutDatabaseReturnsNotImplemented 覆盖无 DB 时各
// 仓方法统一返回 persistence.ErrNotImplemented（内存降级语义）。
func TestPostgresRepositoryWithoutDatabaseReturnsNotImplemented(t *testing.T) {
	repo := NewPostgresRepository(nil)
	ctx := context.Background()

	if _, _, err := repo.ListReactions(ctx, ReactTargetMoment, "m1", "", 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListReactions err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.Aggregate(ctx, ReactTargetMoment, "m1", ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Aggregate err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.AggregateBatch(ctx, ReactTargetMoment, []string{"m1"}, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("AggregateBatch err = %v, want ErrNotImplemented", err)
	}
	if _, _, err := repo.ListReactedTargets(ctx, ReactTargetMoment, "u1", 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListReactedTargets err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.Unreact(ctx, ReactTargetMoment, "m1", "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Unreact err = %v, want ErrNotImplemented", err)
	}
}

// TestDecodeCursorsRejectMalformedTimestamp 覆盖游标里的时间戳无法解析时的
// 非法游标分支（list 与 target 两套命名空间）。
func TestDecodeCursorsRejectMalformedTimestamp(t *testing.T) {
	listToken, err := cursor.Encode(reactionsListCursorNamespace, listCursor{At: "not-a-timestamp", ID: "u1"})
	if err != nil {
		t.Fatalf("encode list cursor: %v", err)
	}
	if _, _, err := decodeListCursor(listToken); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("decodeListCursor(bad at) err = %v, want ErrInvalidCursor", err)
	}
	targetToken, err := cursor.Encode(reactionsTargetCursorNamespace, listCursor{At: "not-a-timestamp", ID: "m1"})
	if err != nil {
		t.Fatalf("encode target cursor: %v", err)
	}
	if _, _, err := decodeTargetCursor(targetToken); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("decodeTargetCursor(bad at) err = %v, want ErrInvalidCursor", err)
	}
	// 空游标是合法「第一页」语义。
	if at, id, err := decodeListCursor(""); at != nil || id != "" || err != nil {
		t.Fatalf("decodeListCursor(empty) = (%v, %q, %v)", at, id, err)
	}
	if at, id, err := decodeTargetCursor(""); at != nil || id != "" || err != nil {
		t.Fatalf("decodeTargetCursor(empty) = (%v, %q, %v)", at, id, err)
	}
}

// TestHandlerListMapsCursorError 覆盖 handler 把 ErrInvalidCursor 映射为 10000
// 的分支（service -> repo 返回非法游标）。
func TestHandlerListMapsCursorError(t *testing.T) {
	svc := NewServiceWithDeps(&g5ReactionsRepo{listErr: ErrInvalidCursor}, []TargetProvider{
		stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}},
	}, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.List, "", g5ListBody)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeInvalidRequest {
		t.Fatalf("cursor error: code = %d, want %d", code, httpx.CodeInvalidRequest)
	}
}

// TestHandlerListMapsUnexpectedError 覆盖 handler 对非业务错误的 10600 兜底。
func TestHandlerListMapsUnexpectedError(t *testing.T) {
	svc := NewServiceWithDeps(&g5ReactionsRepo{listErr: errors.New("boom")}, []TargetProvider{
		stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}},
	}, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.List, "", g5ListBody)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeInternalUnavailable {
		t.Fatalf("unexpected repo error: code = %d, want %d", code, httpx.CodeInternalUnavailable)
	}
}

// TestServiceListRejectsInvalidTargetAndClampsLimit 覆盖 service.List 的非法
// target 提前返回与 pagination.limit > MaxPageSize 的钳制分支。
func TestServiceListRejectsInvalidTargetAndClampsLimit(t *testing.T) {
	ctx := context.Background()
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{
		stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}},
	}, nil, nil)

	if _, err := svc.List(ctx, ListReactionsRequest{TargetType: ReactTarget("bogus"), TargetID: "x"}); !errors.Is(err, ErrReactionTargetNotFound) {
		t.Fatalf("invalid target type err = %v, want ErrReactionTargetNotFound", err)
	}
	if _, err := svc.List(ctx, ListReactionsRequest{TargetType: ReactTargetMoment, TargetID: "m1", Pagination: &PaginationParams{Limit: 1000}}); err != nil {
		t.Fatalf("clamped limit list err = %v", err)
	}
}

// TestServiceResolveTargetAcrossDegradedModes 覆盖 resolveTarget 的两条降级
// 分支：providers map 为 nil（裸 Service）与 provider 缺失某 target kind。
func TestServiceResolveTargetAcrossDegradedModes(t *testing.T) {
	ctx := context.Background()

	bare := &Service{repo: NewInMemoryRepository()}
	if _, err := bare.List(ctx, ListReactionsRequest{TargetType: ReactTargetMoment, TargetID: "m1"}); err != nil {
		t.Fatalf("bare service List err = %v", err)
	}

	withProviders := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{
		stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}},
	}, nil, nil)
	if _, err := withProviders.List(ctx, ListReactionsRequest{TargetType: ReactTargetComment, TargetID: "c1"}); err != nil {
		t.Fatalf("missing-kind provider List err = %v", err)
	}
}

// TestServicePropagatesProviderAndRepoErrors 覆盖 react 的 provider 错误与
// unreact 的仓储错误上抛分支。
func TestServicePropagatesProviderAndRepoErrors(t *testing.T) {
	ctx := context.Background()

	providerErrSvc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{g5ErrTargetProvider{kind: ReactTargetMoment}}, nil, nil)
	if err := providerErrSvc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true}); err == nil || err.Error() != "provider down" {
		t.Fatalf("provider error = %v, want provider down", err)
	}

	repoErrSvc := NewServiceWithDeps(&g5ReactionsRepo{unreactErr: errors.New("boom")}, nil, nil, nil)
	if err := repoErrSvc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: false}); err == nil || err.Error() != "boom" {
		t.Fatalf("unreact repo error = %v, want boom", err)
	}
}
