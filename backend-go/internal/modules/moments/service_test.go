package moments

import (
	"context"
	"errors"
	"strconv"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/types"
)

type stubMomentRepo struct {
	created     MomentItem
	createErr   error
	getResult   MomentItem
	getFound    bool
	getErr      error
	getCalls    int         // Get 调用计数
	getSecond   *MomentItem // 第 2 次起 Get 返回该值（模拟 Update 后 reload 出新版本）
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
	r.getCalls++
	if r.getSecond != nil && r.getCalls >= 2 {
		return *r.getSecond, true, r.getErr
	}
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

// 软删瞬间一律按不存在处理，含作者本人（spec api_spec_moment.md「删除语义」：
// 软删占位只出现在 fetch_created/fetch_favorites）。
func TestServiceOpenTheDoorSoftDeletedNotFoundEvenForOwner(t *testing.T) {
	deleted := ownedMoment()
	deleted.IsDeleted = true
	repo := &stubMomentRepo{getResult: deleted, getFound: true}
	svc := NewServiceWithRepo(repo)
	_, ok, err := svc.OpenTheDoor(momentCtx("u1"), "m1")
	if err != nil || ok {
		t.Fatalf("expected soft-deleted moment to be not found for owner, got ok=%v err=%v", ok, err)
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

func TestServiceCheckCommentableDeletedMoment(t *testing.T) {
	// 软删瞬间按不存在处理（spec api_spec_moment.md 评论挂载；对齐
	// IsReactionTargetVisible 口径，防已删瞬间被评论且计数被污染）。
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, PrivacyLevel: "public", IsDeleted: true}, getFound: true}
	svc := NewServiceWithRepo(repo)
	found, canComment, err := svc.CheckCommentable(momentCtx("u1"), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected found=false for deleted moment, got found=%v", found)
	}
	if canComment {
		t.Fatalf("expected canComment=false, got true")
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

// TestServiceListCreatedEnsuresArrayShape：无图动态的 images/tags/mentions 必须
// 序列化为 [] 而非 null（spec 恒定形状；此前 ListCreated 缺失该步骤，客户端把
// null 渲染成空 src 图片 → 我的瞬间页破图）。
func TestServiceListCreatedEnsuresArrayShape(t *testing.T) {
	repo := &stubMomentRepo{
		listMineRes: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}},
	}
	svc := NewServiceWithRepo(repo)
	result, err := svc.ListCreated(momentCtx("u1"), ListCreatedMomentsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item := result.Items[0]
	if item.Images == nil || len(item.Images) != 0 {
		t.Fatalf("expected images to be empty slice, got %#v", item.Images)
	}
	if item.Tags == nil || item.Mentions == nil {
		t.Fatalf("expected tags/mentions to be empty slices, got tags=%#v mentions=%#v", item.Tags, item.Mentions)
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

// --- mentions 上限与增量通知（S4/M9 修复配套） ---

// stubNotifyWriter 记录 AppendMentioned 的目标与调用次数。
type stubNotifyWriter struct {
	appended map[string]int
}

func (w *stubNotifyWriter) AppendMentionedBatch(_ context.Context, targetUserIDs []string, _ types.UserInfo, _ string, _ string, _ types.MomentSummary, _ string, _ string) (map[string]string, error) {
	if w.appended == nil {
		w.appended = make(map[string]int)
	}
	ids := make(map[string]string, len(targetUserIDs))
	for i, targetUserID := range targetUserIDs {
		w.appended[targetUserID]++
		ids[targetUserID] = "n" + strconv.Itoa(i+1)
	}
	return ids, nil
}

func TestEnsureMentionsLimitBoundary(t *testing.T) {
	if err := ensureMentionsLimit(make([]string, 20)); err != nil {
		t.Fatalf("20 个 mention 应放行, got %v", err)
	}
	if err := ensureMentionsLimit(make([]string, 21)); !errors.Is(err, ErrTooManyMentions) {
		t.Fatalf("21 个 mention 应拒绝, got %v", err)
	}
	if err := ensureMentionsLimit(nil); err != nil {
		t.Fatalf("nil mentions 应放行, got %v", err)
	}
}

func TestServiceCreateRejectsTooManyMentions(t *testing.T) {
	svc := NewServiceWithDeps(&stubMomentRepo{}, nil, nil)
	req := CreateMomentRequest{Description: "hello", Mentions: make([]string, 21)}
	if _, err := svc.Create(momentCtx("u1"), req); !errors.Is(err, ErrTooManyMentions) {
		t.Fatalf("expected ErrTooManyMentions, got %v", err)
	}
}

func TestServiceUpdateRejectsTooManyMentions(t *testing.T) {
	repo := &stubMomentRepo{getResult: ownedMoment(), getFound: true}
	svc := NewServiceWithRepo(repo)
	req := UpdateMomentRequest{MomentID: "m1", Mentions: make([]string, 21)}
	if _, err := svc.Update(momentCtx("u1"), req); !errors.Is(err, ErrTooManyMentions) {
		t.Fatalf("expected ErrTooManyMentions, got %v", err)
	}
}

func TestServiceUpdateNotifiesOnlyNewMentions(t *testing.T) {
	oldItem := ownedMoment()
	oldItem.Mentions = []string{"u2"}
	updatedItem := oldItem
	updatedItem.Mentions = []string{"u2", "u3"}
	notif := &stubNotifyWriter{}
	repo := &stubMomentRepo{getResult: oldItem, getFound: true, getSecond: &updatedItem}
	svc := NewServiceWithDeps(repo, notif, nil)
	if _, err := svc.Update(momentCtx("u1"), UpdateMomentRequest{MomentID: "m1", Mentions: []string{"u2", "u3"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif.appended["u2"] != 0 {
		t.Fatalf("编辑前已存在的 u2 不应被重复通知, got %d", notif.appended["u2"])
	}
	if notif.appended["u3"] != 1 {
		t.Fatalf("新增的 u3 应恰好通知一次, got %d", notif.appended["u3"])
	}
}

func TestServiceCreateNotifiesAllMentions(t *testing.T) {
	created := ownedMoment()
	created.Mentions = []string{"u2", "u3"}
	notif := &stubNotifyWriter{}
	repo := &stubMomentRepo{created: created}
	svc := NewServiceWithDeps(repo, notif, nil)
	if _, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "hello", Mentions: []string{"u2", "u3", "u1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif.appended["u2"] != 1 || notif.appended["u3"] != 1 {
		t.Fatalf("create 应通知全部新增 mention, got %#v", notif.appended)
	}
	if notif.appended["u1"] != 0 {
		t.Fatalf("作者本人不应被通知, got %d", notif.appended["u1"])
	}
}

func TestServiceCreateNotifiesNearbyUsers(t *testing.T) {
	created := ownedMoment()
	created.PrivacyLevel = "public"
	created.User = UserInfo{UserID: "u1", Nickname: "作者"}
	repo := &stubMomentRepo{created: created}

	var targetsCalls int
	targetsFn := func(ctx context.Context, lat, lng float64, radius int, exclude string) ([]NearbyNotifyTarget, error) {
		targetsCalls++
		if exclude != "u1" {
			t.Fatalf("exclude 应为作者 u1, got %q", exclude)
		}
		return []NearbyNotifyTarget{
			{UserID: "u2", Latitude: lat + 0.001, Longitude: lng},
		}, nil
	}
	published := map[string]int{}
	publisher := stubNearbyPublisher{onPublish: func(target string) { published[target]++ }}

	svc := NewServiceWithDeps(repo, &stubNotifyWriter{}, nil)
	svc.SetNearbyNotifyTargetsReader(targetsFn)
	svc.SetNearbyNotifyPublisher(publisher)

	_, err := svc.Create(momentCtx("u1"), CreateMomentRequest{
		Description: "nearby",
		Location:    &Location{Point: types.GeoPoint{Latitude: 35.0976, Longitude: 112.6124}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetsCalls != 1 {
		t.Fatalf("targets fn 应被调用 1 次, got %d", targetsCalls)
	}
	if published["u2"] != 1 {
		t.Fatalf("u2 应收到附近通知, got %#v", published)
	}
}

func TestServiceCreateSkipsNearbyNotifyWithoutLocation(t *testing.T) {
	created := ownedMoment()
	created.PrivacyLevel = "public"
	repo := &stubMomentRepo{created: created}

	targetsCalls := 0
	svc := NewServiceWithDeps(repo, &stubNotifyWriter{}, nil)
	svc.SetNearbyNotifyTargetsReader(func(ctx context.Context, lat, lng float64, radius int, exclude string) ([]NearbyNotifyTarget, error) {
		targetsCalls++
		return nil, nil
	})
	svc.SetNearbyNotifyPublisher(stubNearbyPublisher{onPublish: func(string) {}})

	if _, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "无位置"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetsCalls != 0 {
		t.Fatalf("无位置时不应查询目标, got %d", targetsCalls)
	}
}

type stubNearbyPublisher struct {
	onPublish func(targetUserID string)
}

func (s stubNearbyPublisher) PublishNearbyNewMoment(ctx context.Context, targetUserID, actorID, momentID string, distance int, actorPayload types.UserInfo, momentSummaryPayload types.MomentSummary) error {
	if s.onPublish != nil {
		s.onPublish(targetUserID)
	}
	return nil
}
