package moments

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubMomentRepo struct {
	created     MomentItem
	createErr   error
	getResult   MomentItem
	getFound    bool
	getErr      error
	updateErr   error
	deleteErr   error
	exists      bool
	existsErr   error
	favRepoErr  error
	unfavErr    error
	searchRes   PaginatedMoments
	searchErr   error
	favRes      PaginatedMoments
	favListErr  error
	incComment  error
	decComment  error
	incShares   error
	listMineRes PaginatedMoments
	listMineErr error
	myState     *MyMomentState
}

func (r *stubMomentRepo) Create(_ context.Context, _ string, _ CreateMomentRequest) (MomentItem, error) {
	return r.created, r.createErr
}
func (r *stubMomentRepo) Get(_ context.Context, _ string) (MomentItem, bool, error) {
	return r.getResult, r.getFound, r.getErr
}
func (r *stubMomentRepo) Search(_ context.Context, _ SearchMomentsRequest, _ string) (PaginatedMoments, error) {
	return r.searchRes, r.searchErr
}
func (r *stubMomentRepo) Update(_ context.Context, _ string, _ UpdateMomentRequest) error {
	return r.updateErr
}
func (r *stubMomentRepo) Delete(_ context.Context, _ string) error { return r.deleteErr }
func (r *stubMomentRepo) ListMine(_ context.Context, _ string, _ int, _ string) (PaginatedMoments, error) {
	return r.listMineRes, r.listMineErr
}
func (r *stubMomentRepo) Exists(_ context.Context, _ string) (bool, error) {
	return r.exists, r.existsErr
}
func (r *stubMomentRepo) IncrementCommentCount(_ context.Context, _ string) error {
	return r.incComment
}
func (r *stubMomentRepo) DecrementCommentCount(_ context.Context, _ string) error {
	return r.decComment
}
func (r *stubMomentRepo) IncrementSharesCount(_ context.Context, _ string) error {
	return r.incShares
}
func (r *stubMomentRepo) Favorite(_ context.Context, _ string, _ string) error {
	return r.favRepoErr
}
func (r *stubMomentRepo) Unfavorite(_ context.Context, _ string, _ string) error {
	return r.unfavErr
}
func (r *stubMomentRepo) ListFavorites(_ context.Context, _ string, _ int, _ string) (PaginatedMoments, error) {
	return r.favRes, r.favListErr
}

func (r *stubMomentRepo) ListFavoriteUserIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (r *stubMomentRepo) LoadMyState(ctx context.Context, a, b string) *MyMomentState {
	return r.myState
}

func (r *stubMomentRepo) FetchMomentImageRows(_ context.Context, _ []string) (map[string][]momentImageRow, error) {
	return map[string][]momentImageRow{}, nil
}

func (r *stubMomentRepo) ListFollowingMoments(_ context.Context, _ momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}

func (r *stubMomentRepo) ListNearbyMoments(_ context.Context, _ momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}

func momentCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func ownedMoment() MomentItem {
	return MomentItem{
		MomentID: "m1",
		User:     UserInfo{UserID: "u1"},
	}
}

func TestServiceCreateDelegatesToRepo(t *testing.T) {
	expected := MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}}
	repo := &stubMomentRepo{created: expected}
	svc := NewServiceWithRepo(repo)
	result, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MomentID != "m1" {
		t.Fatalf("expected m1, got %s", result.MomentID)
	}
}

func TestServiceCreatePropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{createErr: errors.New("boom")})
	_, err := svc.Create(momentCtx("u1"), CreateMomentRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceOpenTheDoorReturnsItem(t *testing.T) {
	expected := ownedMoment()
	repo := &stubMomentRepo{getResult: expected, getFound: true}
	svc := NewServiceWithRepo(repo)
	item, ok, err := svc.OpenTheDoor(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || item.MomentID != "m1" {
		t.Fatalf("expected found m1, got ok=%v item=%#v", ok, item)
	}
}

func TestServiceOpenTheDoorNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	_, ok, err := svc.OpenTheDoor(context.Background(), "missing")
	if err != nil || ok {
		t.Fatalf("expected not found, got ok=%v err=%v", ok, err)
	}
}

func TestServiceUpdateRequiresOwner(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "other"}}, getFound: true}
	svc := NewServiceWithRepo(repo)
	_, err := svc.Update(momentCtx("u1"), UpdateMomentRequest{MomentID: "m1"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceUpdateRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	_, err := svc.Update(momentCtx("u1"), UpdateMomentRequest{MomentID: "m1"})
	if !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceUpdateHappyPath(t *testing.T) {
	repo := &stubMomentRepo{
		getResult: ownedMoment(),
		getFound:  true,
	}
	svc := NewServiceWithRepo(repo)
	_, err := svc.Update(momentCtx("u1"), UpdateMomentRequest{MomentID: "m1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeleteRequiresOwner(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "other"}}, getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Delete(momentCtx("u1"), "m1")
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceDeleteRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	err := svc.Delete(momentCtx("u1"), "m1")
	if !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceDeleteHappyPath(t *testing.T) {
	repo := &stubMomentRepo{getResult: ownedMoment(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Delete(momentCtx("u1"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceFavoriteRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	err := svc.Favorite(momentCtx("u1"), "m1")
	if !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceFavoriteHappyPath(t *testing.T) {
	repo := &stubMomentRepo{getResult: ownedMoment(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Favorite(momentCtx("u1"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnfavoriteDelegates(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{getResult: ownedMoment(), getFound: true})
	err := svc.Unfavorite(momentCtx("u1"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceListFavoritesNormalizesPagination(t *testing.T) {
	repo := &stubMomentRepo{favRes: PaginatedMoments{Items: []MomentItem{{MomentID: "m2"}}}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.ListFavorites(momentCtx("u1"), MyFavoritesRequest{
		Pagination: &PaginationParams{Limit: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceSearchDelegates(t *testing.T) {
	repo := &stubMomentRepo{searchRes: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.Search(context.Background(), SearchMomentsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceSearchPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{searchErr: errors.New("boom")})
	_, err := svc.Search(context.Background(), SearchMomentsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceCheckCommentablePublicMoment(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, PrivacyLevel: "public"}, getFound: true}
	svc := NewServiceWithRepo(repo)
	found, canComment, err := svc.CheckCommentable(momentCtx("u_other"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || !canComment {
		t.Fatalf("expected found=true canComment=true, got found=%v canComment=%v", found, canComment)
	}
}

func TestServiceCheckCommentablePrivateMomentOwner(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, PrivacyLevel: "private"}, getFound: true}
	svc := NewServiceWithRepo(repo)
	found, canComment, err := svc.CheckCommentable(momentCtx("u1"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || !canComment {
		t.Fatalf("expected found=true canComment=true, got found=%v canComment=%v", found, canComment)
	}
}

func TestServiceCheckCommentablePrivateMomentOtherUser(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, PrivacyLevel: "private"}, getFound: true}
	svc := NewServiceWithRepo(repo)
	found, canComment, err := svc.CheckCommentable(momentCtx("u_other"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 私有瞬间非拥有者按不存在处理（spec 防枚举，comments/create → 10300）。
	if found {
		t.Fatalf("expected found=false for private moment non-owner, got found=%v", found)
	}
	if canComment {
		t.Fatalf("expected canComment=false, got true")
	}
}

func TestServiceCheckCommentableMomentNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	found, _, err := svc.CheckCommentable(momentCtx("u1"), "missing")
	if err != nil || found {
		t.Fatalf("expected found=false err=nil, got found=%v err=%v", found, err)
	}
}

// ---------- A) Favorite/Unfavorite 错误路径 ----------

func TestServiceFavoritePropagatesRepoError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{getResult: ownedMoment(), getFound: true, favRepoErr: errors.New("boom")})
	err := svc.Favorite(momentCtx("u1"), "m1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestServiceUnfavoriteRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	err := svc.Unfavorite(momentCtx("u1"), "m1")
	if !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("expected ErrMomentNotFound, got %v", err)
	}
}

func TestServiceUnfavoritePropagatesRepoError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{getResult: ownedMoment(), getFound: true, unfavErr: errors.New("boom")})
	err := svc.Unfavorite(momentCtx("u1"), "m1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
}

// ---------- C) 辅助函数 ----------

func TestServiceExistsDelegates(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{exists: true})
	ok, err := svc.Exists(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

func TestServiceExistsPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{existsErr: errors.New("boom")})
	_, err := svc.Exists(context.Background(), "m1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestServiceIncrementCommentCountDelegates(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	err := svc.IncrementCommentCount(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceIncrementSharesCountDelegates(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	err := svc.IncrementSharesCount(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDecrementCommentCountDelegates(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	err := svc.DecrementCommentCount(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------- D) ListMine 和 OpenTheDoor 分支 ----------

func TestServiceListCreatedNormalizesLimit(t *testing.T) {
	repo := &stubMomentRepo{
		listMineRes: PaginatedMoments{Items: []MomentItem{{MomentID: "m1", PrivacyLevel: "public"}}},
	}
	svc := NewServiceWithRepo(repo)
	result, err := svc.ListCreated(momentCtx("u1"), ListCreatedMomentsRequest{
		Pagination: &PaginationParams{Limit: 150},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].PrivacyLevel != "public" {
		t.Fatalf("expected owner view to carry privacy_level, got %#v", result.Items[0])
	}
}

func TestServiceListCreatedDefaultLimit(t *testing.T) {
	repo := &stubMomentRepo{
		listMineRes: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}},
	}
	svc := NewServiceWithRepo(repo)
	result, err := svc.ListCreated(momentCtx("u1"), ListCreatedMomentsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceOpenTheDoorLoadsMyState(t *testing.T) {
	expected := ownedMoment()
	myState := &MyMomentState{IsFavorited: true}
	repo := &stubMomentRepo{getResult: expected, getFound: true, myState: myState}
	svc := NewServiceWithRepo(repo)
	item, ok, err := svc.OpenTheDoor(momentCtx("u1"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected found")
	}
	if !item.IsFavorited {
		t.Fatal("expected IsFavorited=true")
	}
}
