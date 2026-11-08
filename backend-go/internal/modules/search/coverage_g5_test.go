package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func g5SearchCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var env struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response %q: %v", rr.Body.String(), err)
	}
	return env.Code
}

// g5LocationProvider 是 LocationProvider 的固定位置测试替身。
type g5LocationProvider struct {
	point geo.Point
	found bool
}

func (p g5LocationProvider) RecentLocation(context.Context, string) (geo.Point, bool) {
	return p.point, p.found
}

// TestModuleMountsSearchRoutes 覆盖 NewModule 装配与 Mount/MountPublic：
// 公开节点匿名可访问（非法请求体 10000），/api/v1 节点必须挂鉴权。
func TestModuleMountsSearchRoutes(t *testing.T) {
	module := NewModule(nil, nil)
	if module == nil || module.handler == nil || module.Service == nil {
		t.Fatalf("expected wired module, got %#v", module)
	}

	apiRouter := chi.NewRouter()
	module.Mount(apiRouter)
	publicRouter := chi.NewRouter()
	module.MountPublic(publicRouter)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/search/suggest", strings.NewReader("{"))
	publicRouter.ServeHTTP(rr, req)
	if code := g5SearchCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public suggest invalid body: code = %d, want %d", code, httpx.CodeInvalidRequest)
	}

	// /api/v1 组必须挂鉴权：单测未装配 auth service 时会以 10600 拒绝。
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/search/suggest", strings.NewReader("{\"prefix\":\"x\"}"))
	apiRouter.ServeHTTP(rr, req)
	if code := g5SearchCode(t, rr); code != httpx.CodeUnauthorized && code != httpx.CodeInternalUnavailable {
		t.Fatalf("unauthenticated api suggest: code = %d, want auth rejection", code)
	}
}

// TestConstructorFallbacks 覆盖 NewService 与 NewServiceWithSource 的
// nil repo / nil config source 回退分支。
func TestConstructorFallbacks(t *testing.T) {
	svc := NewService(nil)
	if svc == nil || svc.repo == nil || svc.appConfig == nil {
		t.Fatalf("NewService(nil) = %#v", svc)
	}
	if fallback := NewServiceWithSource(nil, nil); fallback == nil || fallback.repo == nil || fallback.appConfig == nil {
		t.Fatalf("NewServiceWithSource(nil, nil) = %#v", fallback)
	}
}

// TestResolveReferencePointUsesRecentLocation 覆盖 reference_point 回退链第 2 级：
// 已登录用户最近位置命中；显式传入时优先于回退。
func TestResolveReferencePointUsesRecentLocation(t *testing.T) {
	svc := NewServiceWithRepo(NewInMemoryRepository())
	svc.SetLocationProvider(g5LocationProvider{point: geo.Point{Latitude: 12.5, Longitude: 34.5}, found: true})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u_g5_search", SessionID: "s1"})

	got := svc.resolveReferencePoint(ctx, nil)
	if got.Latitude != 12.5 || got.Longitude != 34.5 {
		t.Fatalf("reference point = %#v, want recent location (12.5, 34.5)", got)
	}

	explicit := GeoPoint{Latitude: 1, Longitude: 2}
	got = svc.resolveReferencePoint(ctx, &explicit)
	if got.Latitude != 1 || got.Longitude != 2 {
		t.Fatalf("explicit reference point = %#v, want (1, 2)", got)
	}
}

// TestSuggestGeoFilterRejectsInvalidInput 覆盖半径入口的两条短路分支。
func TestSuggestGeoFilterRejectsInvalidInput(t *testing.T) {
	if mins, maxs := suggestGeoFilter(GeoPoint{Latitude: 35, Longitude: 112}, 0); mins != nil || maxs != nil {
		t.Fatalf("radius 0 filter = (%v, %v), want (nil, nil)", mins, maxs)
	}
	if mins, maxs := suggestGeoFilter(GeoPoint{Latitude: 999, Longitude: 0}, 5000); mins != nil || maxs != nil {
		t.Fatalf("invalid point filter = (%v, %v), want (nil, nil)", mins, maxs)
	}
}

// TestPresenceStatusPtrAndDerefString 覆盖 presence 归一化默认分支与 nil 解引用。
func TestPresenceStatusPtrAndDerefString(t *testing.T) {
	if got := presenceStatusPtr("online"); got != "online" {
		t.Fatalf("presenceStatusPtr(online) = %q", got)
	}
	if got := presenceStatusPtr("away"); got != "" {
		t.Fatalf("presenceStatusPtr(away) = %q, want empty", got)
	}
	if got := derefString(nil); got != "" {
		t.Fatalf("derefString(nil) = %q, want empty", got)
	}
	value := "x"
	if got := derefString(&value); got != "x" {
		t.Fatalf("derefString(&x) = %q", got)
	}
}

// TestCompareSuggestItemsTieBreakers 覆盖排序比较器在 tie_breaker 上的两个
// 方向与完全相等分支。
func TestCompareSuggestItemsTieBreakers(t *testing.T) {
	base := scoredSuggestItem{TieBreaker: "b"}
	less := scoredSuggestItem{TieBreaker: "a"}
	if got := compareSuggestItems(less, base); got != -1 {
		t.Fatalf("compare(less, base) = %d, want -1", got)
	}
	if got := compareSuggestItems(base, less); got != 1 {
		t.Fatalf("compare(base, less) = %d, want 1", got)
	}
	if got := compareSuggestItems(base, base); got != 0 {
		t.Fatalf("compare(base, base) = %d, want 0", got)
	}
}

// TestCollectViewerRelationsEmptyTargets 覆盖空目标列表的提前返回分支。
func TestCollectViewerRelationsEmptyTargets(t *testing.T) {
	if err := collectViewerRelations(context.Background(), nil, "viewer", nil,
		map[string]bool{}, map[string]bool{}, map[string]bool{}); err != nil {
		t.Fatalf("collectViewerRelations(empty) err = %v", err)
	}
}

// TestPublicSuggestMapsRepositoryError 覆盖匿名 suggest 在仓储不可用时返回
// 10600 的分支（未接 DB 的 Postgres 仓储走 ErrNotImplemented）。
func TestPublicSuggestMapsRepositoryError(t *testing.T) {
	h := NewHandler(NewService(nil))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/public/v1/search/suggest", strings.NewReader("{\"prefix\":\"g5\"}"))
	h.PublicSuggest(rr, req)
	if code := g5SearchCode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("public suggest repo error: code = %d, want %d", code, httpx.CodeInternalUnavailable)
	}
}

// TestPostgresIT_SuggestFillsViewerRelations 真实库覆盖登录视角 suggest 的
// relation 批量填充（走 collectViewerRelations 的正向/反向两条查询）：
// 查看者关注的人、被拉黑的人分别拿到 is_following / is_blocked / is_followed_by。
func TestPostgresIT_SuggestFillsViewerRelations(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		viewer := testdb.InsertUser(t, txFactory, "u_g5_search_viewer", "G5Viewer")
		followed := testdb.InsertUser(t, txFactory, "u_g5_search_followed", "G5SearchAlpha")
		blocked := testdb.InsertUser(t, txFactory, "u_g5_search_blocked", "G5SearchBeta")
		stranger := testdb.InsertUser(t, txFactory, "u_g5_search_plain", "G5SearchGamma")
		for _, userID := range []string{viewer, followed, blocked, stranger} {
			if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT DO NOTHING", userID); err != nil {
				t.Fatalf("insert user_settings %s: %v", userID, err)
			}
		}
		if err := qdb.Exec(ctx, "INSERT INTO user_relations (user_id, target_user_id, is_following, is_blocked) VALUES ($1, $2, TRUE, FALSE), ($1, $3, TRUE, TRUE), ($2, $1, TRUE, FALSE)",
			viewer, followed, blocked); err != nil {
			t.Fatalf("insert user_relations: %v", err)
		}

		repo := NewPostgresRepository(txFactory)
		viewerCtx := platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: viewer, SessionID: "s1"})
		resp, err := repo.Suggest(viewerCtx, SearchSuggestRequest{Prefix: "G5Search", Types: []string{"user"}})
		if err != nil {
			t.Fatalf("Suggest: %v", err)
		}
		relations := map[string]*SearchSuggestUserRelation{}
		for _, item := range resp.Items {
			payload, ok := item.Payload.(SearchSuggestUserPayload)
			if !ok {
				continue
			}
			relations[payload.User.UserID] = payload.Relation
		}
		alpha, ok := relations[followed]
		if !ok || alpha == nil || !alpha.IsFollowing || alpha.IsBlocked {
			t.Fatalf("followed user relation = %#v (present=%v), want following only", alpha, ok)
		}
		// followed 是双向关注：正向行 + 反向行都存在 -> IsFollowedBy 也应为 true。
		if !alpha.IsFollowedBy {
			t.Fatalf("followed user relation = %#v, want is_followed_by true", alpha)
		}
		beta, ok := relations[blocked]
		if !ok || beta == nil || !beta.IsFollowing || !beta.IsBlocked || beta.IsFollowedBy {
			t.Fatalf("blocked user relation = %#v (present=%v), want following+blocked", beta, ok)
		}
		gamma, ok := relations[stranger]
		if !ok || gamma != nil {
			t.Fatalf("stranger relation = %#v (present=%v), want nil", gamma, ok)
		}
	})
}
