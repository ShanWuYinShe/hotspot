package moments

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

type stubActorProvider struct {
	info actor.Info
	err  error
}

func (p stubActorProvider) GetActorInfo(context.Context, string) (actor.Info, error) {
	return p.info, p.err
}

type branchServiceRepo struct {
	*stubMomentRepo
	favoriteUserIDs    []string
	favoriteUserIDsErr error
	listFollowingErr   error
	listNearbyErr      error
}

func (r *branchServiceRepo) ListFavoriteUserIDs(context.Context, string, int) ([]string, error) {
	if r.favoriteUserIDsErr != nil {
		return nil, r.favoriteUserIDsErr
	}
	return r.favoriteUserIDs, nil
}

func (r *branchServiceRepo) ListFollowingMoments(context.Context, momentsQuery) (PaginatedMoments, error) {
	if r.listFollowingErr != nil {
		return PaginatedMoments{}, r.listFollowingErr
	}
	return PaginatedMoments{}, nil
}

func (r *branchServiceRepo) ListNearbyMoments(context.Context, momentsQuery) (PaginatedMoments, error) {
	if r.listNearbyErr != nil {
		return PaginatedMoments{}, r.listNearbyErr
	}
	return PaginatedMoments{}, nil
}

type reloadRepo struct {
	*stubMomentRepo
	calls       int
	second      MomentItem
	secondFound bool
	secondErr   error
}

func (r *reloadRepo) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	r.calls++
	if r.calls >= 2 {
		return r.second, r.secondFound, r.secondErr
	}
	return r.stubMomentRepo.Get(ctx, momentID)
}

type purgerRepo struct {
	*stubMomentRepo
	err error
}

func (r *purgerRepo) PurgeUserData(context.Context, string) error { return r.err }

type favLimitRepo struct {
	*stubMomentRepo
	limit  int
	cursor string
}

func (r *favLimitRepo) ListFavorites(_ context.Context, _ string, limit int, cursor string) (PaginatedMoments, error) {
	r.limit = limit
	r.cursor = cursor
	return PaginatedMoments{}, nil
}

type imageRowsRepo struct {
	*stubMomentRepo
	rows map[string][]momentImageRow
	err  error
}

func (r *imageRowsRepo) FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error) {
	return r.rows, r.err
}

type stubLocationProvider struct {
	pt    geo.Point
	found bool
}

func (p stubLocationProvider) RecentLocation(context.Context, string) (geo.Point, bool) {
	return p.pt, p.found
}

type failingNearbyPublisher struct{}

func (failingNearbyPublisher) PublishNearbyNewMomentPreChecked(context.Context, string, string, string, int, types.UserInfo, types.MomentSummary) error {
	return errors.New("publish down")
}

// TestServiceUpdateTxAndReloadBranches 覆盖 Update 的事务执行、事务失败与
// 提交后重载失败/不存在分支。
func TestServiceUpdateTxAndReloadBranches(t *testing.T) {
	desc := "new"
	req := UpdateMomentRequest{MomentID: "m1", Description: &desc}
	ctx := momentCtx("u1")

	// 事务成功（runner 直接执行 updateFn）
	txRepo := &stubMomentRepo{getResult: ownedMoment(), getFound: true}
	txSvc := NewServiceWithDepsAndImagesAndConfig(txRepo, nil, nil, nil, nil, func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	item, err := txSvc.Update(ctx, req)
	if err != nil || item.MomentID != "m1" {
		t.Fatalf("tx update: %#v err=%v", item, err)
	}

	// 事务失败 → 错误向上传播
	errRepo := &stubMomentRepo{getResult: ownedMoment(), getFound: true}
	errSvc := NewServiceWithDepsAndImagesAndConfig(errRepo, nil, nil, nil, nil, func(context.Context, func(context.Context) error) error {
		return errMomentFake
	})
	if _, err := errSvc.Update(ctx, req); !errors.Is(err, errMomentFake) {
		t.Fatalf("tx error must propagate, got %v", err)
	}

	// 无事务 + 仓储更新失败
	repo := &stubMomentRepo{getResult: ownedMoment(), getFound: true, updateErr: errMomentFake}
	if _, err := NewServiceWithRepo(repo).Update(ctx, req); !errors.Is(err, errMomentFake) {
		t.Fatalf("repo update error, got %v", err)
	}

	// 提交后重载失败
	reloadErr := &reloadRepo{stubMomentRepo: &stubMomentRepo{getResult: ownedMoment(), getFound: true}, secondErr: errMomentFake}
	if _, err := NewServiceWithRepo(reloadErr).Update(ctx, req); !errors.Is(err, errMomentFake) {
		t.Fatalf("reload error, got %v", err)
	}

	// 提交后重载不到（并发删除）→ ErrMomentNotFound
	reloadMissing := &reloadRepo{stubMomentRepo: &stubMomentRepo{getResult: ownedMoment(), getFound: true}}
	if _, err := NewServiceWithRepo(reloadMissing).Update(ctx, req); !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("reload missing, got %v", err)
	}
}

// TestServiceFillReactionsBatchOwnerMomentBranches 覆盖作者视角批量聚合的
// nil 聚合器 / 聚合失败 / 成功 / 空列表分支。
func TestServiceFillReactionsBatchOwnerMomentBranches(t *testing.T) {
	ctx := context.Background()
	items := []OwnerMomentItem{
		{MomentItem: MomentItem{MomentID: "m1"}},
		{MomentItem: MomentItem{MomentID: "m2"}},
	}

	// nil 聚合器：全部填空数组（非 nil）
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.fillReactionsBatchOwnerMoment(ctx, items, "viewer")
	for _, item := range items {
		if item.Reactions == nil || len(item.Reactions) != 0 {
			t.Fatalf("nil aggregator must fill empty reactions: %#v", item.Reactions)
		}
	}

	// 聚合失败：全部填空数组
	failItems := []OwnerMomentItem{{MomentItem: MomentItem{MomentID: "m1"}}}
	svc.SetReactionAggregator(&momentsReactionAggregator{err: errors.New("boom")})
	svc.fillReactionsBatchOwnerMoment(ctx, failItems, "viewer")
	if failItems[0].Reactions == nil || len(failItems[0].Reactions) != 0 {
		t.Fatalf("aggregator error must degrade: %#v", failItems[0].Reactions)
	}

	// 成功：按 moment 填充并透传 viewer
	agg := &momentsReactionAggregator{result: map[string][]Reaction{
		"m1": {{Emoji: "👍", Count: 3}},
	}}
	okItems := []OwnerMomentItem{{MomentItem: MomentItem{MomentID: "m1"}}, {MomentItem: MomentItem{MomentID: "m2"}}}
	svc2 := NewServiceWithRepo(&stubMomentRepo{})
	svc2.SetReactionAggregator(agg)
	svc2.fillReactionsBatchOwnerMoment(ctx, okItems, "viewer")
	if len(okItems[0].Reactions) != 1 || okItems[0].Reactions[0].Emoji != "👍" {
		t.Fatalf("reactions not filled: %#v", okItems[0].Reactions)
	}
	if okItems[1].Reactions == nil || len(okItems[1].Reactions) != 0 {
		t.Fatalf("missing moment must be empty: %#v", okItems[1].Reactions)
	}
	if len(agg.viewers) != 1 || agg.viewers[0] != "viewer" {
		t.Fatalf("viewer not passed: %v", agg.viewers)
	}

	// 空列表：不调用聚合器
	emptyAgg := &momentsReactionAggregator{}
	svc3 := NewServiceWithRepo(&stubMomentRepo{})
	svc3.SetReactionAggregator(emptyAgg)
	svc3.fillReactionsBatchOwnerMoment(ctx, nil, "viewer")
	if len(emptyAgg.calls) != 0 {
		t.Fatalf("empty list must not aggregate: %v", emptyAgg.calls)
	}
}

// TestServiceBroadcastMomentDeletedToFavorites 覆盖删除后收藏者广播的成功/
// 仓储失败/超上限截断分支（删除主流程不因广播失败而失败）。
func TestServiceBroadcastMomentDeletedToFavorites(t *testing.T) {
	previousHub := realtime.DefaultHub()
	realtime.SetDefaultHub(realtime.NewHub())
	t.Cleanup(func() { realtime.SetDefaultHub(previousHub) })

	repo := &branchServiceRepo{
		stubMomentRepo:  &stubMomentRepo{getResult: ownedMoment(), getFound: true},
		favoriteUserIDs: []string{"u1", "u2", "u3"},
	}
	svc := NewServiceWithRepo(repo)
	if err := svc.Delete(momentCtx("u1"), "m1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// 收藏者查询失败：主流程仍成功（尽力而为）。
	repo.favoriteUserIDsErr = errMomentFake
	if err := svc.Delete(momentCtx("u1"), "m1"); err != nil {
		t.Fatalf("delete with broadcast error must succeed, got %v", err)
	}

	// 超上限截断：不 panic、主流程成功。
	repo.favoriteUserIDsErr = nil
	repo.favoriteUserIDs = make([]string, momentDeleteBroadcastFanoutCap+1)
	for i := range repo.favoriteUserIDs {
		repo.favoriteUserIDs[i] = "fanout_user_" + time.Unix(int64(i), 0).UTC().Format("150405")
	}
	if err := svc.Delete(momentCtx("u1"), "m1"); err != nil {
		t.Fatalf("delete with fanout cap: %v", err)
	}

	// 直接调用错误路径返回 error。
	repo.favoriteUserIDsErr = errMomentFake
	if err := svc.broadcastMomentDeletedToFavorites(context.Background(), "m1", "u1", "now"); !errors.Is(err, errMomentFake) {
		t.Fatalf("expected broadcast error, got %v", err)
	}
}

// TestServiceTagAssignmentBranches 覆盖服务层标签关联能力的支持/不支持分支。
func TestServiceTagAssignmentBranches(t *testing.T) {
	ctx := context.Background()
	unsupported := NewServiceWithRepo(&stubMomentRepo{})
	if _, err := unsupported.HasTagAssignments(ctx, "x"); !errors.Is(err, errTagAssignmentStoreUnsupported) {
		t.Fatalf("unsupported HasTagAssignments: %v", err)
	}
	if err := unsupported.DeleteTagAssignments(ctx, "x"); !errors.Is(err, errTagAssignmentStoreUnsupported) {
		t.Fatalf("unsupported DeleteTagAssignments: %v", err)
	}

	repo := NewInMemoryRepository()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "tagged", Tags: []string{"旅行"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	svc := NewServiceWithRepo(repo)
	has, err := svc.HasTagAssignments(ctx, "旅行")
	if err != nil || !has {
		t.Fatalf("HasTagAssignments: %v %v", has, err)
	}
	if err := svc.DeleteTagAssignments(ctx, "旅行"); err != nil {
		t.Fatalf("DeleteTagAssignments: %v", err)
	}
	has, err = svc.HasTagAssignments(ctx, "旅行")
	if err != nil || has {
		t.Fatalf("tag should be gone: %v %v", has, err)
	}
	stored, _, _ := repo.Get(ctx, created.MomentID)
	if len(stored.Tags) != 0 {
		t.Fatalf("tag assignments must be removed: %#v", stored.Tags)
	}
}

// TestServicePurgeUserDataBranches 覆盖账户注销级联的服务层三态。
func TestServicePurgeUserDataBranches(t *testing.T) {
	ctx := context.Background()
	// 非 purger 仓储 → nil（无跨模块行可清）
	if err := NewServiceWithRepo(NewInMemoryRepository()).PurgeUserData(ctx, "u1"); err != nil {
		t.Fatalf("non-purger: %v", err)
	}
	// purger 成功 / 失败
	if err := NewServiceWithRepo(&purgerRepo{stubMomentRepo: &stubMomentRepo{}}).PurgeUserData(ctx, "u1"); err != nil {
		t.Fatalf("purger success: %v", err)
	}
	if err := NewServiceWithRepo(&purgerRepo{stubMomentRepo: &stubMomentRepo{}, err: errMomentFake}).PurgeUserData(ctx, "u1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("purger error: %v", err)
	}
}

// TestServiceValidateImageOwnershipBranches 覆盖图片归属校验的配置与解析分支。
func TestServiceValidateImageOwnershipBranches(t *testing.T) {
	ctx := context.Background()
	// 空列表直接放行
	if err := NewServiceWithRepo(&stubMomentRepo{}).validateImageOwnership(ctx, "u1", nil); err != nil {
		t.Fatalf("empty ids: %v", err)
	}
	// appConfig 存在但未配置上限 → 不限制；images 未注入 → 放行
	svc := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, nil, momentsAppConfigStub{cfg: appconfig.AppConfig{}}, nil)
	if err := svc.validateImageOwnership(ctx, "u1", []string{"f1", "f2"}); err != nil {
		t.Fatalf("unlimited config: %v", err)
	}
	// 配置上限且超限 → ErrTooManyImages
	limited := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, nil, momentsAppConfigStub{cfg: appconfig.AppConfig{
		Limits: appconfig.Limits{MaxMomentImages: 1},
	}}, nil)
	if err := limited.validateImageOwnership(ctx, "u1", []string{"f1", "f2"}); !errors.Is(err, ErrTooManyImages) {
		t.Fatalf("over limit: %v", err)
	}
	// 解析成功 → 放行
	resolved := NewServiceWithDepsAndImages(&stubMomentRepo{}, nil, nil, momentsImageResolver{assets: map[string]ImageAsset{"f1": {FileID: "f1"}}})
	if err := resolved.validateImageOwnership(ctx, "u1", []string{"f1"}); err != nil {
		t.Fatalf("resolved ownership: %v", err)
	}
	// 解析失败 → ErrImagesNotOwned
	denied := NewServiceWithDepsAndImages(&stubMomentRepo{}, nil, nil, momentsImageResolver{err: errors.New("denied")})
	if err := denied.validateImageOwnership(ctx, "u1", []string{"f1"}); !errors.Is(err, ErrImagesNotOwned) {
		t.Fatalf("denied ownership: %v", err)
	}
}

// TestServiceHydrateImagesBatchBranches 覆盖批量图片 hydrate 的短路、
// 已取回行（presign only）与仓储批量查询三条路径。
func TestServiceHydrateImagesBatchBranches(t *testing.T) {
	ctx := context.Background()
	resolver := momentsImageResolver{assets: map[string]ImageAsset{"f1": {FileID: "f1", StorageKey: "uploads/1.jpg"}}}

	// images 未注入 → 不修改
	noImages := NewServiceWithRepo(&stubMomentRepo{})
	items := []MomentItem{{MomentID: "m1"}}
	noImages.hydrateImagesBatch(ctx, items)
	if items[0].Images != nil {
		t.Fatalf("nil resolver must not fill: %#v", items[0].Images)
	}

	// 空列表 → 直接返回
	withImages := NewServiceWithDepsAndImages(&stubMomentRepo{}, nil, nil, resolver)
	withImages.hydrateImagesBatch(ctx, nil)

	// 行已随详情带回 → 只 presign
	loaded := []MomentItem{{
		MomentID:      "m1",
		imagesFetched: true,
		imageRows:     []momentImageRow{{FileID: "f1", StorageKey: "uploads/1.jpg", Width: 10, Height: 20}},
	}}
	withImages.hydrateImagesBatch(ctx, loaded)
	if len(loaded[0].Images) != 1 || loaded[0].Images[0].URL != "https://cdn.example/uploads/1.jpg" {
		t.Fatalf("loaded rows not presigned: %#v", loaded[0].Images)
	}

	// 未取回 → 走仓储批量查询
	fetchedRepo := &imageRowsRepo{
		stubMomentRepo: &stubMomentRepo{},
		rows:           map[string][]momentImageRow{"m2": {{FileID: "f1", StorageKey: "uploads/1.jpg", MimeType: "image/jpeg"}}},
	}
	batchSvc := NewServiceWithDepsAndImages(fetchedRepo, nil, nil, resolver)
	batch := []MomentItem{{MomentID: "m2"}}
	batchSvc.hydrateImagesBatch(ctx, batch)
	if len(batch[0].Images) != 1 || batch[0].Images[0].MimeType == nil || *batch[0].Images[0].MimeType != "image/jpeg" {
		t.Fatalf("batch rows not hydrated: %#v", batch[0].Images)
	}
}

// TestServiceBuildMomentActorAndMentionFallback 覆盖 actor 解析的 nil/成功/失败
// 分支，以及失败时按 fallback actor 继续通知的行为。
func TestServiceBuildMomentActorAndMentionFallback(t *testing.T) {
	ctx := context.Background()
	if got, err := NewServiceWithRepo(&stubMomentRepo{}).buildMomentActor(ctx, " u1 "); err != nil || got.UserID != "u1" || got.Nickname != "u1" {
		t.Fatalf("fallback actor: %#v err=%v", got, err)
	}
	okSvc := NewServiceWithDeps(&stubMomentRepo{}, nil, stubActorProvider{info: actor.Info{UserID: "u1", Nickname: "昵称", AvatarURL: "a"}})
	got, err := okSvc.buildMomentActor(ctx, "u1")
	if err != nil || got.Nickname != "昵称" || got.AvatarURL == nil || *got.AvatarURL != "a" {
		t.Fatalf("actor info: %#v err=%v", got, err)
	}
	failSvc := NewServiceWithDeps(&stubMomentRepo{}, nil, stubActorProvider{err: errMomentFake})
	if _, err := failSvc.buildMomentActor(ctx, "u1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("actor error: %v", err)
	}

	// actor 查询失败不得阻断 Create 的 mention 通知（回退 userID/nickname）。
	created := ownedMoment()
	created.Mentions = []string{"u2"}
	notif := &stubNotifyWriter{}
	repo := &stubMomentRepo{created: created}
	svc := NewServiceWithDeps(repo, notif, stubActorProvider{err: errMomentFake})
	if _, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "hi", Mentions: []string{"u2"}}); err != nil {
		t.Fatalf("create with actor failure: %v", err)
	}
	if notif.appended["u2"] != 1 {
		t.Fatalf("mention notification must still be sent, got %#v", notif.appended)
	}
}

// TestServiceFillAuthorRelationsEdgeBranches 覆盖已取回标记、空作者 id 与
// 无可查询作者的短路。
func TestServiceFillAuthorRelationsEdgeBranches(t *testing.T) {
	reader := &fakeFollowRelationReader{result: map[string]UserRelationView{"u1": {IsFollowing: true}}}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetFollowRelationReader(reader)

	items := []MomentItem{
		{MomentID: "m1", User: UserInfo{UserID: "u1"}, authorRelationFetched: true},
		{MomentID: "m2", User: UserInfo{UserID: ""}},
	}
	svc.fillAuthorRelations(context.Background(), items, "viewer")
	if len(reader.gotAuthors) != 0 {
		t.Fatalf("no author ids should be queried, got %v", reader.gotAuthors)
	}
	if items[0].AuthorRelation != nil || items[1].AuthorRelation != nil {
		t.Fatalf("skipped items must stay nil: %#v %#v", items[0].AuthorRelation, items[1].AuthorRelation)
	}
}

// TestServiceIsReactionTargetVisibleBranches 覆盖拉黑查询失败 fail-closed 与
// 拉黑不可见分支。
func TestServiceIsReactionTargetVisibleBranches(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "author"}, PrivacyLevel: "public"}, getFound: true}
	svc := NewServiceWithRepo(repo)
	ctx := momentCtx("caller")

	svc.SetBlockChecker(func(string, string) (bool, error) { return false, errMomentFake })
	found, visible, err := svc.IsReactionTargetVisible(ctx, "m1")
	if !found || visible || !errors.Is(err, errMomentFake) {
		t.Fatalf("block error must fail closed: found=%v visible=%v err=%v", found, visible, err)
	}

	svc.SetBlockChecker(func(string, string) (bool, error) { return true, nil })
	found, visible, err = svc.IsReactionTargetVisible(ctx, "m1")
	if !found || visible || err != nil {
		t.Fatalf("blocked target must be invisible: found=%v visible=%v err=%v", found, visible, err)
	}

	svc.SetBlockChecker(nil)
	found, visible, err = svc.IsReactionTargetVisible(ctx, "m1")
	if !found || !visible || err != nil {
		t.Fatalf("public target must be visible: found=%v visible=%v err=%v", found, visible, err)
	}
}

// TestServiceEnsureDescriptionLengthBranches 覆盖 AppConfig 上限与默认上限。
func TestServiceEnsureDescriptionLengthBranches(t *testing.T) {
	ctx := context.Background()
	configured := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, nil, momentsAppConfigStub{cfg: appconfig.AppConfig{
		Limits: appconfig.Limits{MomentDescriptionMaxLength: 3},
	}}, nil)
	if err := configured.ensureDescriptionLength(ctx, "abcd"); !errors.Is(err, ErrDescriptionTooLong) {
		t.Fatalf("configured limit: %v", err)
	}
	if err := configured.ensureDescriptionLength(ctx, "abc"); err != nil {
		t.Fatalf("within configured limit: %v", err)
	}
	// 未配置上限（0）→ 回退默认 2000 runes
	fallback := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, nil, momentsAppConfigStub{cfg: appconfig.AppConfig{}}, nil)
	if err := fallback.ensureDescriptionLength(ctx, "ok"); err != nil {
		t.Fatalf("default limit short text: %v", err)
	}
	long := make([]rune, defaultMomentDescriptionMaxRunes+1)
	for i := range long {
		long[i] = '长'
	}
	if err := fallback.ensureDescriptionLength(ctx, string(long)); !errors.Is(err, ErrDescriptionTooLong) {
		t.Fatalf("default limit long text: %v", err)
	}
}

// TestServiceNearbyNotifyBranches 覆盖附近通知的目标查询失败、发布失败与
// 非公开瞬间跳过（均不得阻断创建主流程）。
func TestServiceNearbyNotifyBranches(t *testing.T) {
	loc := &Location{Point: types.GeoPoint{Latitude: 35.0, Longitude: 112.0}}
	created := ownedMoment()
	created.PrivacyLevel = "public"
	created.User = UserInfo{UserID: "u1", Nickname: "作者"}

	// 目标查询失败 → 静默跳过，创建成功
	targetFail := NewServiceWithDeps(&stubMomentRepo{created: created}, nil, nil)
	targetFail.SetNearbyNotifyTargetsReader(func(context.Context, float64, float64, int, string, int) ([]NearbyNotifyTarget, error) {
		return nil, errMomentFake
	})
	targetFail.SetNearbyNotifyPublisher(failingNearbyPublisher{})
	if _, err := targetFail.Create(momentCtx("u1"), CreateMomentRequest{Description: "x", Location: loc}); err != nil {
		t.Fatalf("target failure must not block create: %v", err)
	}

	// 发布失败 → 静默跳过，创建成功
	publishFail := NewServiceWithDeps(&stubMomentRepo{created: created}, nil, nil)
	publishFail.SetNearbyNotifyTargetsReader(func(context.Context, float64, float64, int, string, int) ([]NearbyNotifyTarget, error) {
		return []NearbyNotifyTarget{{UserID: "u2", Latitude: 35.001, Longitude: 112.0}}, nil
	})
	publishFail.SetNearbyNotifyPublisher(failingNearbyPublisher{})
	if _, err := publishFail.Create(momentCtx("u1"), CreateMomentRequest{Description: "x", Location: loc}); err != nil {
		t.Fatalf("publish failure must not block create: %v", err)
	}

	// 私密瞬间 → 不触发附近通知
	privateCreated := ownedMoment()
	privateCreated.PrivacyLevel = "private"
	published := 0
	privateSvc := NewServiceWithDeps(&stubMomentRepo{created: privateCreated}, nil, nil)
	privateSvc.SetNearbyNotifyTargetsReader(func(context.Context, float64, float64, int, string, int) ([]NearbyNotifyTarget, error) {
		published++
		return nil, nil
	})
	privateSvc.SetNearbyNotifyPublisher(stubNearbyPublisher{})
	if _, err := privateSvc.Create(momentCtx("u1"), CreateMomentRequest{Description: "x", Location: loc}); err != nil {
		t.Fatalf("private create: %v", err)
	}
	if published != 0 {
		t.Fatalf("private moment must not notify nearby, got %d", published)
	}
}

// TestServiceDiscoveryErrorPaths 覆盖 fetch_following / fetch_nearby 的仓储错误。
func TestServiceDiscoveryErrorPaths(t *testing.T) {
	repo := &branchServiceRepo{stubMomentRepo: &stubMomentRepo{}, listFollowingErr: errMomentFake, listNearbyErr: errMomentFake}
	svc := NewServiceWithRepo(repo)
	if _, err := svc.FetchFollowing(momentCtx("u1"), FetchFollowingMomentsRequest{}); !errors.Is(err, errMomentFake) {
		t.Fatalf("fetch following error: %v", err)
	}
	if _, err := svc.FetchNearby(momentCtx("u1"), FetchNearbyMomentsRequest{ReferencePoint: &GeoPoint{Latitude: 35, Longitude: 112}}); !errors.Is(err, errMomentFake) {
		t.Fatalf("fetch nearby error: %v", err)
	}
}

// TestServiceOpenTheDoorPrivateAndDeleteBranches 覆盖私密瞬间详情可见性、
// 重复删除幂等与删除仓储失败。
func TestServiceOpenTheDoorPrivateAndDeleteBranches(t *testing.T) {
	repo := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, PrivacyLevel: "private"}, getFound: true}
	svc := NewServiceWithRepo(repo)
	if _, ok, err := svc.OpenTheDoor(momentCtx("u1"), "m1"); err != nil || !ok {
		t.Fatalf("private owner detail: ok=%v err=%v", ok, err)
	}
	if _, ok, err := svc.OpenTheDoor(momentCtx("u2"), "m1"); err != nil || ok {
		t.Fatalf("private non-owner detail must be hidden: ok=%v err=%v", ok, err)
	}

	// 重复删除（已软删）→ no-op 成功
	deleted := &stubMomentRepo{getResult: MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, IsDeleted: true}, getFound: true}
	if err := NewServiceWithRepo(deleted).Delete(momentCtx("u1"), "m1"); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}

	// 仓储删除失败 → 包装错误
	failing := &stubMomentRepo{getResult: ownedMoment(), getFound: true, deleteErr: errMomentFake}
	if err := NewServiceWithRepo(failing).Delete(momentCtx("u1"), "m1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("delete repo error: %v", err)
	}
}

// TestServiceInteractableAndListBranches 覆盖交互前读取失败、收藏列表 limit
// 裁剪与游标透传、以及 reactions 批量填充的边界。
func TestServiceInteractableAndListBranches(t *testing.T) {
	ctx := momentCtx("u1")
	repo := &stubMomentRepo{getErr: errMomentFake}
	if err := NewServiceWithRepo(repo).Favorite(ctx, "m1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("favorite get error: %v", err)
	}

	favRepo := &favLimitRepo{stubMomentRepo: &stubMomentRepo{}}
	svc := NewServiceWithRepo(favRepo)
	if _, err := svc.ListFavorites(ctx, MyFavoritesRequest{Pagination: &types.PaginationParams{Limit: 500, Cursor: "cursor-x"}}); err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if favRepo.limit != types.MaxPageSize {
		t.Fatalf("limit clamp: got %d", favRepo.limit)
	}
	if favRepo.cursor != "cursor-x" {
		t.Fatalf("cursor passthrough: got %q", favRepo.cursor)
	}

	// reactions 已取回 → 不再聚合
	agg := &momentsReactionAggregator{}
	reactionSvc := NewServiceWithRepo(&stubMomentRepo{})
	reactionSvc.SetReactionAggregator(agg)
	fetchedItems := []MomentItem{{MomentID: "m1", reactionsFetched: true}}
	reactionSvc.fillReactionsBatch(context.Background(), fetchedItems, "viewer")
	if len(agg.calls) != 0 {
		t.Fatalf("fetched reactions must skip aggregation: %v", agg.calls)
	}

	// nil 聚合器 → 填空数组
	plainItems := []MomentItem{{MomentID: "m2"}}
	NewServiceWithRepo(&stubMomentRepo{}).fillReactionsBatch(context.Background(), plainItems, "viewer")
	if plainItems[0].Reactions == nil || len(plainItems[0].Reactions) != 0 {
		t.Fatalf("nil aggregator must fill empty: %#v", plainItems[0].Reactions)
	}
}

// TestServiceSmallHelperBranches 覆盖 NotificationContext 错误、DeleteAsAdmin
// 读取错误、nil 入参短路与相对位置填充。
func TestServiceSmallHelperBranches(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{getErr: errMomentFake})
	if _, ok, err := svc.NotificationContext(context.Background(), "m1"); !errors.Is(err, errMomentFake) || ok {
		t.Fatalf("notification context error: ok=%v err=%v", ok, err)
	}
	if err := svc.DeleteAsAdmin(context.Background(), "m1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("delete as admin get error: %v", err)
	}

	svc.fillReactions(context.Background(), nil, "viewer")
	svc.fillAuthorRelation(context.Background(), nil, "viewer")

	loc := &Location{}
	fillRelativeLocation(loc, 120, 45)
	if loc.Distance == nil || *loc.Distance != 120 || loc.BearingDegrees == nil || *loc.BearingDegrees != 45 {
		t.Fatalf("fill relative location: %#v", loc)
	}
	fillRelativeLocation(nil, 1, 1) // 不 panic
}

// ── 发现流（discovery.go）辅助分支 ──

// TestResolveReferencePointBranches 覆盖 reference_point 回退链的四级分支。
func TestResolveReferencePointBranches(t *testing.T) {
	ctx := context.Background()
	svc := NewServiceWithRepo(&stubMomentRepo{})

	// 1. 显式合法坐标直接使用
	explicit := &GeoPoint{Latitude: 10, Longitude: 20}
	if got := svc.resolveReferencePoint(ctx, explicit); got == nil || got.Latitude != 10 || got.Longitude != 20 {
		t.Fatalf("explicit point: %#v", got)
	}
	// 2. 显式非法坐标 → nil
	if got := svc.resolveReferencePoint(ctx, &GeoPoint{Latitude: 999}); got != nil {
		t.Fatalf("invalid explicit point must be nil: %#v", got)
	}
	// 3. 已登录 + 用户最近位置
	svc.SetLocationProvider(stubLocationProvider{pt: geo.Point{Latitude: 31.2, Longitude: 121.5}, found: true})
	got := svc.resolveReferencePoint(momentCtx("u1"), nil)
	if got == nil || got.Latitude != 31.2 || got.Longitude != 121.5 {
		t.Fatalf("recent location fallback: %#v", got)
	}
	// 4. 最近位置不存在 → AppConfig 默认位置
	svc.SetLocationProvider(stubLocationProvider{found: false})
	svc.appConfig = momentsAppConfigStub{cfg: appconfig.AppConfig{
		Defaults: appconfig.Defaults{DefaultLocation: appconfig.Location{Point: appconfig.GeoPoint{Latitude: 35.1, Longitude: 112.6}}},
	}}
	got = svc.resolveReferencePoint(momentCtx("u1"), nil)
	if got == nil || got.Latitude != 35.1 || got.Longitude != 112.6 {
		t.Fatalf("appconfig fallback: %#v", got)
	}
	// 5. AppConfig 缺失 → 平台默认锚点
	bare := NewServiceWithRepo(&stubMomentRepo{})
	got = bare.resolveReferencePoint(ctx, nil)
	if got == nil || got.Latitude != geo.DefaultFallbackLatitude || got.Longitude != geo.DefaultFallbackLongitude {
		t.Fatalf("platform fallback: %#v", got)
	}
}

// TestNormalizeFetchLimitBranches 覆盖发现流分页大小的默认/上限/正常分支。
func TestNormalizeFetchLimitBranches(t *testing.T) {
	cases := []struct {
		name string
		in   *PaginationParams
		want int
	}{
		{"nil", nil, 20},
		{"zero", &PaginationParams{Limit: 0}, 20},
		{"negative", &PaginationParams{Limit: -3}, 20},
		{"normal", &PaginationParams{Limit: 50}, 50},
		{"over max", &PaginationParams{Limit: 101}, 100},
	}
	for _, tc := range cases {
		if got := normalizeFetchLimit(tc.in); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

// TestCompareNearbyCandidatesBranches 覆盖附近候选全序比较的各分支。
func TestCompareNearbyCandidatesBranches(t *testing.T) {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	left := momentCandidate{Item: MomentItem{MomentID: "m2"}, Score: 5, Distance: 100, CreatedAt: base}
	if got := compareNearbyCandidates(left, left); got != 0 {
		t.Fatalf("equal: %d", got)
	}
	cases := []struct {
		name  string
		left  momentCandidate
		right momentCandidate
		want  int
	}{
		// 距离/分值分支返回差值（负=更靠前），时间/ID 分支返回 ±1。
		{"closer", momentCandidate{Distance: 50}, left, -50},
		{"farther", momentCandidate{Distance: 150}, left, 50},
		{"higher score", momentCandidate{Distance: 100, Score: 9}, left, -4},
		{"lower score", momentCandidate{Distance: 100, Score: 1}, left, 4},
		{"newer", momentCandidate{Distance: 100, Score: 5, CreatedAt: base.Add(time.Minute)}, left, -1},
		{"older", momentCandidate{Distance: 100, Score: 5, CreatedAt: base.Add(-time.Minute)}, left, 1},
		{"larger id", momentCandidate{Item: MomentItem{MomentID: "m3"}, Distance: 100, Score: 5, CreatedAt: base}, left, -1},
		{"smaller id", momentCandidate{Item: MomentItem{MomentID: "m1"}, Distance: 100, Score: 5, CreatedAt: base}, left, 1},
	}
	for _, tc := range cases {
		if got := compareNearbyCandidates(tc.left, tc.right); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

// TestDecodeFetchCursorBranches 覆盖发现流游标解码的合法/非法分支。
func TestDecodeFetchCursorBranches(t *testing.T) {
	if cur, err := decodeFetchCursor(""); err != nil || cur != nil {
		t.Fatalf("empty cursor: %#v err=%v", cur, err)
	}
	if _, err := decodeFetchCursor("garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage cursor: %v", err)
	}
	if _, err := decodeFetchCursor(encodeFetchCursor(itemCursor{CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("missing id must fail: %v", err)
	}
	if _, err := decodeFetchCursor(encodeFetchCursor(itemCursor{ID: "m1", CreatedAt: "bad-time"})); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid time must fail: %v", err)
	}
	valid := encodeFetchCursor(itemCursor{ID: "m1", Score: 3, Distance: 7, CreatedAt: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)})
	cur, err := decodeFetchCursor(valid)
	if err != nil || cur == nil || cur.ID != "m1" || cur.Score != 3 || cur.Distance != 7 {
		t.Fatalf("valid cursor: %#v err=%v", cur, err)
	}
}

// TestFindNearbyCursorStartDiscoveryBranches 覆盖发现流分页定位游标起点。
func TestFindNearbyCursorStartDiscoveryBranches(t *testing.T) {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	items := []momentCandidate{
		{Item: MomentItem{MomentID: "m1"}, Distance: 10, CreatedAt: base},
		{Item: MomentItem{MomentID: "m2"}, Distance: 20, CreatedAt: base},
	}
	if got := findNearbyCursorStart(items, nil); got != 0 {
		t.Fatalf("nil cursor: %d", got)
	}
	// 非法 CreatedAt 解析为零值时间：首项（同距同分）被判为早于游标，
	// 第二项因距离更大而返回其下标。
	if got := findNearbyCursorStart(items, &itemCursor{ID: "m1", Distance: 10, CreatedAt: "bad"}); got != 1 {
		t.Fatalf("bad created_at cursor: %d", got)
	}
	hit := &itemCursor{ID: "m1", Distance: 10, CreatedAt: base.Format(time.RFC3339Nano)}
	if got := findNearbyCursorStart(items, hit); got != 1 {
		t.Fatalf("hit cursor: %d", got)
	}
	before := &itemCursor{ID: "mx", Distance: 5, CreatedAt: base.Format(time.RFC3339Nano)}
	if got := findNearbyCursorStart(items, before); got != 0 {
		t.Fatalf("cursor before list: %d", got)
	}
	after := &itemCursor{ID: "mx", Distance: 99, CreatedAt: base.Format(time.RFC3339Nano)}
	if got := findNearbyCursorStart(items, after); got != len(items) {
		t.Fatalf("cursor after list: %d", got)
	}
}

// TestPaginateNearbyAndFollowingBranches 覆盖两种发现流分页的游标生成与非法游标。
func TestPaginateNearbyAndFollowingBranches(t *testing.T) {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	items := []momentCandidate{
		{Item: MomentItem{MomentID: "m1"}, Distance: 10, CreatedAt: base},
		{Item: MomentItem{MomentID: "m2"}, Distance: 20, CreatedAt: base},
		{Item: MomentItem{MomentID: "m3"}, Distance: 30, CreatedAt: base},
	}
	page, err := paginateNearbyMoments(items, 2, "")
	if err != nil || len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("nearby page: %#v err=%v", page, err)
	}
	if _, err := paginateNearbyMoments(items, 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("nearby bad cursor: %v", err)
	}
	// limit 大于候选数 → 全量且无游标
	all, err := paginateNearbyMoments(items, 10, "")
	if err != nil || len(all.Items) != 3 || all.NextCursor != nil {
		t.Fatalf("nearby all: %#v err=%v", all, err)
	}

	if _, err := paginateFollowingMoments(items, 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("following bad cursor: %v", err)
	}
	// 注意：仓库层已按 LIMIT+1 拉取，分页函数按 limit 裁剪并以页尾生成游标。
	followPage, err := paginateFollowingMoments(items, 2, "")
	if err != nil || len(followPage.Items) != 2 || followPage.NextCursor == nil {
		t.Fatalf("following page: %#v err=%v", followPage, err)
	}
}

// TestCompareFollowingCandidatesBranches 覆盖关注流排序的 ID 兜底分支。
func TestCompareFollowingCandidatesBranches(t *testing.T) {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	left := momentCandidate{Item: MomentItem{MomentID: "m2"}, CreatedAt: base}
	if got := compareFollowingCandidates(left, left); got != 0 {
		t.Fatalf("equal: %d", got)
	}
	if got := compareFollowingCandidates(momentCandidate{Item: MomentItem{MomentID: "m3"}, CreatedAt: base}, left); got != -1 {
		t.Fatalf("larger id: %d", got)
	}
	if got := compareFollowingCandidates(momentCandidate{Item: MomentItem{MomentID: "m1"}, CreatedAt: base}, left); got != 1 {
		t.Fatalf("smaller id: %d", got)
	}
	if got := compareFollowingCandidates(momentCandidate{Item: MomentItem{MomentID: "m3"}, CreatedAt: base.Add(time.Minute)}, left); got != -1 {
		t.Fatalf("newer: %d", got)
	}
	if got := compareFollowingCandidates(momentCandidate{Item: MomentItem{MomentID: "m3"}, CreatedAt: base.Add(-time.Minute)}, left); got != 1 {
		t.Fatalf("older: %d", got)
	}
}
