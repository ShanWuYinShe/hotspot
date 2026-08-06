package moments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

// momentErrRepo 以错误 repo 驱动 handler 的 10600 分支。
type momentErrRepo struct {
	repo *InMemoryRepository
	err  error
}

func (r *momentErrRepo) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	return r.repo.Create(ctx, userID, req)
}
func (r *momentErrRepo) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	item, ok, err := r.repo.Get(ctx, momentID)
	if r.err != nil {
		return MomentItem{}, false, r.err
	}
	return item, ok, err
}
func (r *momentErrRepo) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	return r.repo.Search(ctx, req)
}
func (r *momentErrRepo) Update(ctx context.Context, userID string, req EditMomentRequest) error {
	return r.repo.Update(ctx, userID, req)
}
func (r *momentErrRepo) Delete(ctx context.Context, momentID string) error {
	return r.repo.Delete(ctx, momentID)
}
func (r *momentErrRepo) ListMine(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	if r.err != nil {
		return PaginatedMoments{}, r.err
	}
	return r.repo.ListMine(ctx, userID, limit, cursor)
}
func (r *momentErrRepo) Exists(ctx context.Context, momentID string) (bool, error) {
	return r.repo.Exists(ctx, momentID)
}
func (r *momentErrRepo) IncrementCommentCount(ctx context.Context, momentID string) error {
	return r.repo.IncrementCommentCount(ctx, momentID)
}
func (r *momentErrRepo) DecrementCommentCount(ctx context.Context, momentID string) error {
	return r.repo.DecrementCommentCount(ctx, momentID)
}
func (r *momentErrRepo) IncrementSharesCount(ctx context.Context, momentID string) error {
	return r.repo.IncrementSharesCount(ctx, momentID)
}
func (r *momentErrRepo) Favorite(ctx context.Context, userID string, momentID string) error {
	return r.repo.Favorite(ctx, userID, momentID)
}
func (r *momentErrRepo) Unfavorite(ctx context.Context, userID string, momentID string) error {
	return r.repo.Unfavorite(ctx, userID, momentID)
}
func (r *momentErrRepo) ListFavorites(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	return r.repo.ListFavorites(ctx, userID, limit, cursor)
}
func (r *momentErrRepo) ListFavoriteUserIDs(ctx context.Context, momentID string) ([]string, error) {
	return r.repo.ListFavoriteUserIDs(ctx, momentID)
}
func (r *momentErrRepo) LoadMyState(ctx context.Context, momentID, userID string) *MyMomentState {
	return r.repo.LoadMyState(ctx, momentID, userID)
}
func (r *momentErrRepo) FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error) {
	return r.repo.FetchMomentImageRows(ctx, momentIDs)
}

func decodeMomentCode(t *testing.T, rr *httptest.ResponseRecorder) int {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Code
}

func TestMomentHandlerErrorPaths(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&momentErrRepo{repo: NewInMemoryRepository()}))

	// OpenTheDoor：参数错误
	req := httptest.NewRequest(http.MethodPost, "/get_detail", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.OpenTheDoor(rr, req)
	if code := decodeMomentCode(t, rr); code != 10000 {
		t.Fatalf("empty moment id: expected 10000, got %d", code)
	}

	// OpenTheDoor：不存在 → 10300
	req = httptest.NewRequest(http.MethodPost, "/get_detail", strings.NewReader(`{"moment_id":"missing"}`))
	rr = httptest.NewRecorder()
	handler.OpenTheDoor(rr, req)
	if code := decodeMomentCode(t, rr); code != 10300 {
		t.Fatalf("missing moment: expected 10300, got %d", code)
	}

	// OpenTheDoor：服务错误 → 10600
	handler2 := NewHandler(NewServiceWithRepo(&momentErrRepo{repo: NewInMemoryRepository(), err: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/get_detail", strings.NewReader(`{"moment_id":"m1"}`))
	rr = httptest.NewRecorder()
	handler2.OpenTheDoor(rr, req)
	if code := decodeMomentCode(t, rr); code != 10600 {
		t.Fatalf("service error: expected 10600, got %d", code)
	}

	// ListMine：服务错误 → 10600
	handler3 := NewHandler(NewServiceWithRepo(&momentErrRepo{repo: NewInMemoryRepository(), err: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/list_my", strings.NewReader(`{}`))
	rr = httptest.NewRecorder()
	handler3.ListMine(rr, req)
	if code := decodeMomentCode(t, rr); code != 10600 {
		t.Fatalf("list error: expected 10600, got %d", code)
	}
}

func TestCreateMomentRejectsBadCoordinates(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hi","location":{"point":{"latitude":999,"longitude":999}}}`))
	req.Header.Set("Idempotency-Key", "coords1234coords1234coords1234coords")
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	if code := decodeMomentCode(t, rr); code != 10000 {
		t.Fatalf("bad coordinates: expected 10000, got %d", code)
	}
}

func TestCreateMomentRejectsOverlongDescription(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	long := strings.Repeat("啊", momentDescriptionMaxRunes+1)
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"`+long+`"}`))
	req.Header.Set("Idempotency-Key", "overlong1234overlong1234over1234")
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	if code := decodeMomentCode(t, rr); code != 10000 {
		t.Fatalf("overlong description: expected 10000, got %d", code)
	}
}

func TestMomentUpdateAndDeleteErrorPaths(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := context.Background()

	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Update 不存在 → 10300
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"moment_id":"missing","description":"x"}`))
	req = req.WithContext(principalMoments("u1"))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	if code := decodeMomentCode(t, rr); code != 10300 {
		t.Fatalf("update missing: expected 10300, got %d", code)
	}

	// Update 非 owner → 10200
	req = httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"moment_id":"`+created.MomentID+`","description":"x"}`))
	req = req.WithContext(principalMoments("u2"))
	rr = httptest.NewRecorder()
	handler.Update(rr, req)
	if code := decodeMomentCode(t, rr); code != 10200 {
		t.Fatalf("update foreign: expected 10200, got %d", code)
	}

	// Update 参数缺失 → 10000
	req = httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{}`))
	req = req.WithContext(principalMoments("u1"))
	rr = httptest.NewRecorder()
	handler.Update(rr, req)
	if code := decodeMomentCode(t, rr); code != 10000 {
		t.Fatalf("update empty: expected 10000, got %d", code)
	}

	// Delete 不存在 → 10300
	req = httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"moment_id":"missing"}`))
	req = req.WithContext(principalMoments("u1"))
	rr = httptest.NewRecorder()
	handler.Delete(rr, req)
	if code := decodeMomentCode(t, rr); code != 10300 {
		t.Fatalf("delete missing: expected 10300, got %d", code)
	}
}

func principalMoments(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: "sess_1"})
}

func TestMomentFavoriteErrorPaths(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))

	// 空 moment_id → 10000
	req := httptest.NewRequest(http.MethodPost, "/favorite", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.Favorite(rr, req)
	if code := decodeMomentCode(t, rr); code != 10000 {
		t.Fatalf("empty favorite: expected 10000, got %d", code)
	}

	// 不存在的 moment → 10300
	req = httptest.NewRequest(http.MethodPost, "/favorite", strings.NewReader(`{"moment_id":"missing","active":true}`))
	req = req.WithContext(principalMoments("u1"))
	rr = httptest.NewRecorder()
	handler.Favorite(rr, req)
	if code := decodeMomentCode(t, rr); code != 10300 {
		t.Fatalf("favorite missing: expected 10300, got %d", code)
	}

	// ListFavorites 参数错误 → 10000
	req = httptest.NewRequest(http.MethodPost, "/list_favorites", strings.NewReader(`not-json`))
	req = req.WithContext(principalMoments("u1"))
	rr = httptest.NewRecorder()
	handler.ListFavorites(rr, req)
	if code := decodeMomentCode(t, rr); code != 10000 {
		t.Fatalf("list favorites malformed: expected 10000, got %d", code)
	}
}

func TestMomentDeletePermissionDenied(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 非 owner 删除 → 10200
	req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"moment_id":"`+created.MomentID+`"}`))
	req = req.WithContext(principalMoments("u2"))
	rr := httptest.NewRecorder()
	handler.Delete(rr, req)
	if code := decodeMomentCode(t, rr); code != 10200 {
		t.Fatalf("foreign delete: expected 10200, got %d", code)
	}
}
