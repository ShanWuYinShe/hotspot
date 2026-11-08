package moments

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// handlerBranchRepo 在内存仓储之上注入各方法的错误，用于驱动 handler 的
// 错误码映射分支（每个 case 都断言响应 code，而非仅执行）。
type handlerBranchRepo struct {
	*InMemoryRepository
	createErr        error
	updateErr        error
	deleteErr        error
	searchErr        error
	listMineErr      error
	listFavoritesErr error
	listFollowingErr error
	listNearbyErr    error
	favoriteErr      error
	unfavoriteErr    error
	getErr           error
}

func newHandlerBranchRepo() *handlerBranchRepo {
	return &handlerBranchRepo{InMemoryRepository: NewInMemoryRepository()}
}

func (r *handlerBranchRepo) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	if r.createErr != nil {
		return MomentItem{}, r.createErr
	}
	return r.InMemoryRepository.Create(ctx, userID, req)
}

func (r *handlerBranchRepo) Update(ctx context.Context, userID string, req UpdateMomentRequest) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.InMemoryRepository.Update(ctx, userID, req)
}

func (r *handlerBranchRepo) Delete(ctx context.Context, momentID string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.InMemoryRepository.Delete(ctx, momentID)
}

func (r *handlerBranchRepo) Search(ctx context.Context, req SearchMomentsRequest, viewerID string) (PaginatedMoments, error) {
	if r.searchErr != nil {
		return PaginatedMoments{}, r.searchErr
	}
	return r.InMemoryRepository.Search(ctx, req, viewerID)
}

func (r *handlerBranchRepo) ListMine(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	if r.listMineErr != nil {
		return PaginatedMoments{}, r.listMineErr
	}
	return r.InMemoryRepository.ListMine(ctx, userID, limit, cursor)
}

func (r *handlerBranchRepo) ListFavorites(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	if r.listFavoritesErr != nil {
		return PaginatedMoments{}, r.listFavoritesErr
	}
	return r.InMemoryRepository.ListFavorites(ctx, userID, limit, cursor)
}

func (r *handlerBranchRepo) ListFollowingMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	if r.listFollowingErr != nil {
		return PaginatedMoments{}, r.listFollowingErr
	}
	return r.InMemoryRepository.ListFollowingMoments(ctx, query)
}

func (r *handlerBranchRepo) ListNearbyMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	if r.listNearbyErr != nil {
		return PaginatedMoments{}, r.listNearbyErr
	}
	return r.InMemoryRepository.ListNearbyMoments(ctx, query)
}

func (r *handlerBranchRepo) Favorite(ctx context.Context, userID, momentID string) error {
	if r.favoriteErr != nil {
		return r.favoriteErr
	}
	return r.InMemoryRepository.Favorite(ctx, userID, momentID)
}

func (r *handlerBranchRepo) Unfavorite(ctx context.Context, userID, momentID string) error {
	if r.unfavoriteErr != nil {
		return r.unfavoriteErr
	}
	return r.InMemoryRepository.Unfavorite(ctx, userID, momentID)
}

func (r *handlerBranchRepo) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	if r.getErr != nil {
		return MomentItem{}, false, r.getErr
	}
	return r.InMemoryRepository.Get(ctx, momentID)
}

// validIdempotencyKey 生成符合契约（32-64 位 [A-Za-z0-9_-]）的测试幂等键。
func validIdempotencyKey(seed string) string {
	key := seed + strings.Repeat("0", 64)
	return key[:32]
}

func callMomentHandler(t *testing.T, fn func(http.ResponseWriter, *http.Request), body string, userID string, idempotencyKey string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/moments", strings.NewReader(body))
	if userID != "" {
		req = req.WithContext(principalMoments(userID))
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	rr := httptest.NewRecorder()
	fn(rr, req)
	return rr
}

func seedHandlerMoment(t *testing.T, repo *InMemoryRepository, owner string) MomentItem {
	t.Helper()
	item, err := repo.Create(context.Background(), owner, CreateMomentRequest{
		Description: "seed",
		Location: &Location{LocationName: "南天门", Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 35.0, Longitude: 112.0}},
	})
	if err != nil {
		t.Fatalf("seed moment: %v", err)
	}
	return item
}

// TestHandlerCreateValidationAndErrorMappings 覆盖 moments/create 的参数校验与
// service sentinel error → 业务码映射（10000/10300/10400/10600）。
func TestHandlerCreateValidationAndErrorMappings(t *testing.T) {
	key := validIdempotencyKey("create-branch-0001")
	validLoc := `"location":{"location_name":"南天门","point":{"latitude":35.0,"longitude":112.0}}`
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))

	cases := []struct {
		name string
		body string
		key  string
		want httpx.Code
	}{
		{"malformed json", "not-json", key, httpx.CodeInvalidRequest},
		{"missing idempotency key", `{"description":"x",` + validLoc + "}", "", httpx.CodeInvalidRequest},
		{"missing description", "{" + validLoc + "}", key, httpx.CodeInvalidRequest},
		{"missing location", `{"description":"x"}`, key, httpx.CodeInvalidRequest},
		{"invalid privacy", `{"description":"x","privacy_level":"secret",` + validLoc + "}", key, httpx.CodeInvalidRequest},
		{"bad coordinates", `{"description":"x","location":{"point":{"latitude":999,"longitude":0}}}`, key, httpx.CodeInvalidRequest},
		{"too many tags", `{"description":"x","tags":["1","2","3","4","5","6","7","8","9","10","11"],` + validLoc + "}", key, httpx.CodeInvalidRequest},
		{"invalid tag", `{"description":"x","tags":["` + strings.Repeat("长", 31) + `"],` + validLoc + "}", key, httpx.CodeInvalidRequest},
		{"too many mentions", `{"description":"x","mentions":["` + strings.Join(manyMentionIDs(), `","`) + `"],` + validLoc + "}", key, httpx.CodeInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := callMomentHandler(t, handler.Create, tc.body, "u1", tc.key)
			if code := testdb.DecodeAPICode(t, rr); code != tc.want {
				t.Fatalf("expected code %d, got %d", tc.want, code)
			}
		})
	}

	// image 数量上限（AppConfig 配置优先）
	limited := NewHandler(NewServiceWithDepsAndImagesAndConfig(NewInMemoryRepository(), nil, nil, nil, momentsAppConfigStub{cfg: appconfig.AppConfig{
		Limits: appconfig.Limits{MaxMomentImages: 1},
	}}, nil))
	rr := callMomentHandler(t, limited.Create, `{"description":"x","image_file_ids":["f1","f2"],`+validLoc+"}", "u1", validIdempotencyKey("create-branch-0002"))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("too many images: expected 10000, got %d", code)
	}

	// 图片归属校验失败 → 10300
	denied := NewHandler(NewServiceWithDepsAndImages(NewInMemoryRepository(), nil, nil, momentsImageResolver{err: errors.New("denied")}))
	rr = callMomentHandler(t, denied.Create, `{"description":"x","image_file_ids":["f1"],`+validLoc+"}", "u1", validIdempotencyKey("create-branch-0003"))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("images not owned: expected 10300, got %d", code)
	}

	// 被屏蔽标签 → 10400
	blockedRepo := NewInMemoryRepository()
	blockedRepo.SetBlockedTags("bad")
	blocked := NewHandler(NewServiceWithRepo(blockedRepo))
	rr = callMomentHandler(t, blocked.Create, `{"description":"x","tags":["bad"],`+validLoc+"}", "u1", validIdempotencyKey("create-branch-0004"))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeConflict {
		t.Fatalf("blocked tag: expected 10400, got %d", code)
	}

	// 仓储故障 → 10600
	failing := newHandlerBranchRepo()
	failing.createErr = errors.New("db down")
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(failing)).Create, `{"description":"x",`+validLoc+"}", "u1", validIdempotencyKey("create-branch-0005"))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("repo failure: expected 10600, got %d", code)
	}
}

// TestHandlerUpdateErrorMappings 覆盖 moments/update 的入参防御与全部 sentinel
// error 映射（10000/10300/10400/10600）。
func TestHandlerUpdateErrorMappings(t *testing.T) {
	repo := newHandlerBranchRepo()
	item := seedHandlerMoment(t, repo.InMemoryRepository, "owner")
	handler := NewHandler(NewServiceWithRepo(repo))
	body := func(extra string) string {
		return `{"moment_id":"` + item.MomentID + `"` + extra + "}"
	}

	cases := []struct {
		name string
		body string
		want httpx.Code
	}{
		{"malformed json", "not-json", httpx.CodeInvalidRequest},
		{"bad coordinates", body(`,"location":{"point":{"latitude":999,"longitude":0}}`), httpx.CodeInvalidRequest},
		{"invalid privacy", body(`,"privacy_level":"secret"`), httpx.CodeInvalidRequest},
		{"too many tags", body(`,"tags":["1","2","3","4","5","6","7","8","9","10","11"]`), httpx.CodeInvalidRequest},
		{"invalid tag", body(`,"tags":["` + strings.Repeat("长", 31) + `"]`), httpx.CodeInvalidRequest},
		{"too many mentions", body(`,"mentions":["` + strings.Join(manyMentionIDs(), `","`) + `"]`), httpx.CodeInvalidRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := callMomentHandler(t, handler.Update, tc.body, "owner", "")
			if code := testdb.DecodeAPICode(t, rr); code != tc.want {
				t.Fatalf("expected code %d, got %d", tc.want, code)
			}
		})
	}

	// 图片数量超限（AppConfig）
	limitedRepo := newHandlerBranchRepo()
	limitedItem := seedHandlerMoment(t, limitedRepo.InMemoryRepository, "owner")
	limited := NewHandler(NewServiceWithDepsAndImagesAndConfig(limitedRepo, nil, nil, nil, momentsAppConfigStub{cfg: appconfig.AppConfig{
		Limits: appconfig.Limits{MaxMomentImages: 1},
	}}, nil))
	rr := callMomentHandler(t, limited.Update, `{"moment_id":"`+limitedItem.MomentID+`","image_file_ids":["f1","f2"]}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("too many update images: expected 10000, got %d", code)
	}

	// 图片归属失败 → 10300
	denied := NewHandler(NewServiceWithDepsAndImages(repo, nil, nil, momentsImageResolver{err: errors.New("denied")}))
	rr = callMomentHandler(t, denied.Update, `{"moment_id":"`+item.MomentID+`","image_file_ids":["f1"]}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("update images not owned: expected 10300, got %d", code)
	}

	// 被屏蔽标签 → 10400
	blockedRepo := newHandlerBranchRepo()
	blockedRepo.SetBlockedTags("bad")
	blockedItem := seedHandlerMoment(t, blockedRepo.InMemoryRepository, "owner")
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(blockedRepo)).Update, `{"moment_id":"`+blockedItem.MomentID+`","tags":["bad"]}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeConflict {
		t.Fatalf("update blocked tag: expected 10400, got %d", code)
	}

	// 仓储故障 → 10600
	failing := newHandlerBranchRepo()
	failingItem := seedHandlerMoment(t, failing.InMemoryRepository, "owner")
	failing.updateErr = errors.New("db down")
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(failing)).Update, `{"moment_id":"`+failingItem.MomentID+`","description":"new"}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("update repo failure: expected 10600, got %d", code)
	}
}

func manyMentionIDs() []string {
	out := make([]string, 21)
	for i := range out {
		out[i] = "mention_user_" + strings.Repeat("x", i+1)
	}
	return out
}

// TestHandlerListAndDeleteGenericErrors 覆盖 list_created / fetch_favorites /
// delete 的服务端错误出口（10600）。
func TestHandlerListAndDeleteGenericErrors(t *testing.T) {
	repo := newHandlerBranchRepo()
	item := seedHandlerMoment(t, repo.InMemoryRepository, "owner")
	repo.listMineErr = errors.New("list mine down")
	repo.listFavoritesErr = errors.New("list fav down")
	repo.deleteErr = errors.New("delete down")
	handler := NewHandler(NewServiceWithRepo(repo))

	rr := callMomentHandler(t, handler.ListCreated, `{}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("list created: expected 10600, got %d", code)
	}
	rr = callMomentHandler(t, handler.ListFavorites, `{}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("list favorites: expected 10600, got %d", code)
	}
	rr = callMomentHandler(t, handler.Delete, `{"moment_id":"`+item.MomentID+`"}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("delete: expected 10600, got %d", code)
	}
}

// TestHandlerFavoriteUnfavoriteErrorPaths 覆盖 active=false 的 Unfavorite 方法名
// 分支与收藏仓储故障。
func TestHandlerFavoriteUnfavoriteErrorPaths(t *testing.T) {
	repo := newHandlerBranchRepo()
	item := seedHandlerMoment(t, repo.InMemoryRepository, "owner")
	handler := NewHandler(NewServiceWithRepo(repo))

	rr := callMomentHandler(t, handler.Favorite, `{"moment_id":"missing","active":false}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("unfavorite missing: expected 10300, got %d", code)
	}
	repo.favoriteErr = errors.New("favorite down")
	rr = callMomentHandler(t, handler.Favorite, `{"moment_id":"`+item.MomentID+`","active":true}`, "owner", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("favorite repo failure: expected 10600, got %d", code)
	}
}

// TestHandlerSearchDecodeAndServiceErrors 覆盖 moments/search 的 JSON 解码失败
// 与仓储故障。
func TestHandlerSearchDecodeAndServiceErrors(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	rr := callMomentHandler(t, handler.Search, "not-json", "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("search malformed: expected 10000, got %d", code)
	}
	repo := newHandlerBranchRepo()
	repo.searchErr = errors.New("search down")
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(repo)).Search, `{"keyword":"hello"}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("search service failure: expected 10600, got %d", code)
	}
}

// TestHandlerFetchFollowingErrors 覆盖 fetch_following 的解码失败与仓储故障。
func TestHandlerFetchFollowingErrors(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	rr := callMomentHandler(t, handler.FetchFollowing, "not-json", "u1", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("fetch following malformed: expected 10000, got %d", code)
	}
	repo := newHandlerBranchRepo()
	repo.listFollowingErr = errors.New("following down")
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(repo)).FetchFollowing, `{}`, "u1", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("fetch following failure: expected 10600, got %d", code)
	}
}

// TestHandlerFetchNearbyErrors 覆盖 fetch_nearby 的参数校验、非法游标与仓储故障。
func TestHandlerFetchNearbyErrors(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	rr := callMomentHandler(t, handler.FetchNearby, "not-json", "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("fetch nearby malformed: expected 10000, got %d", code)
	}
	rr = callMomentHandler(t, handler.FetchNearby, `{"reference_point":{"latitude":999,"longitude":0}}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("invalid reference point: expected 10000, got %d", code)
	}

	repo := newHandlerBranchRepo()
	repo.listNearbyErr = errors.New("nearby down")
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(repo)).FetchNearby, `{"reference_point":{"latitude":35,"longitude":112}}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("fetch nearby failure: expected 10600, got %d", code)
	}

	repo.listNearbyErr = ErrInvalidCursor
	rr = callMomentHandler(t, NewHandler(NewServiceWithRepo(repo)).FetchNearby, `{"reference_point":{"latitude":35,"longitude":112}}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("invalid cursor: expected 10000, got %d", code)
	}
}

// TestPublicHandlerErrorBranches 覆盖匿名节点的解码/不存在/服务端错误映射。
func TestPublicHandlerErrorBranches(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))

	// PublicOpenTheDoor
	rr := callMomentHandler(t, handler.PublicOpenTheDoor, "not-json", "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public open malformed: expected 10000, got %d", code)
	}
	rr = callMomentHandler(t, handler.PublicOpenTheDoor, `{}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public open empty id: expected 10000, got %d", code)
	}
	rr = callMomentHandler(t, handler.PublicOpenTheDoor, `{"moment_id":"missing"}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("public open missing: expected 10300, got %d", code)
	}

	// PublicSearch
	rr = callMomentHandler(t, handler.PublicSearch, "not-json", "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public search malformed: expected 10000, got %d", code)
	}
	rr = callMomentHandler(t, handler.PublicSearch, `{}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public search no criteria: expected 10000, got %d", code)
	}

	// PublicFetchNearby
	rr = callMomentHandler(t, handler.PublicFetchNearby, "not-json", "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public nearby malformed: expected 10000, got %d", code)
	}
	rr = callMomentHandler(t, handler.PublicFetchNearby, `{"reference_point":{"latitude":999,"longitude":0}}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("public nearby invalid ref: expected 10000, got %d", code)
	}

	// 服务端故障 → 10600（Get/Search/ListNearby 三路）
	failing := newHandlerBranchRepo()
	failing.getErr = errors.New("get down")
	failing.searchErr = errors.New("search down")
	failing.listNearbyErr = errors.New("nearby down")
	failingHandler := NewHandler(NewServiceWithRepo(failing))
	rr = callMomentHandler(t, failingHandler.PublicOpenTheDoor, `{"moment_id":"m1"}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("public open service failure: expected 10600, got %d", code)
	}
	rr = callMomentHandler(t, failingHandler.PublicSearch, `{"keyword":"hello"}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("public search service failure: expected 10600, got %d", code)
	}
	rr = callMomentHandler(t, failingHandler.PublicFetchNearby, `{"reference_point":{"latitude":35,"longitude":112}}`, "", "")
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("public nearby service failure: expected 10600, got %d", code)
	}
}
