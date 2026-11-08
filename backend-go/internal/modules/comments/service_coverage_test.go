package comments

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/types"
)

// coverReactionAggregator 记录调用参数并返回预置聚合结果。
type coverReactionAggregator struct {
	calls    int
	ids      []string
	viewerID string
	result   map[string][]Reaction
	err      error
}

func (a *coverReactionAggregator) AggregateComments(_ context.Context, ids []string, viewerID string) (map[string][]Reaction, error) {
	a.calls++
	a.ids = append([]string(nil), ids...)
	a.viewerID = viewerID
	return a.result, a.err
}

// coverTopRepliesRepo 覆盖 fillTopReplies 的批量查询：记录 limit/调用次数并
// 返回预置的 parentID -> replies 映射。
type coverTopRepliesRepo struct {
	*stubCommentRepo
	result    map[string][]CommentItem
	err       error
	calls     int
	limit     int
	parentIDs []string
}

func (r *coverTopRepliesRepo) ListTopRepliesByParentIDs(_ context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error) {
	r.calls++
	r.limit = limit
	r.parentIDs = append([]string(nil), parentIDs...)
	return r.result, r.err
}

// coverSpyListRepo 记录 List/ListReplies 收到的分页参数。
type coverSpyListRepo struct {
	*stubCommentRepo
	listLimit     int
	listCursor    string
	repliesLimit  int
	repliesCursor string
}

func (r *coverSpyListRepo) List(ctx context.Context, momentID string, limit int, cursor string) (PaginatedComments, error) {
	r.listLimit, r.listCursor = limit, cursor
	return r.stubCommentRepo.List(ctx, momentID, limit, cursor)
}

func (r *coverSpyListRepo) ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error) {
	r.repliesLimit, r.repliesCursor = limit, cursor
	return r.stubCommentRepo.ListReplies(ctx, commentID, limit, cursor)
}

// replyWithoutAuthorRepo 的父评论没有作者（User.UserID 为空），用于覆盖
// notifyReplyTarget 的 empty-target 提前返回分支。
type replyWithoutAuthorRepo struct{ *stubCommentRepo }

func (r replyWithoutAuthorRepo) Get(context.Context, string) (CommentItem, bool, error) {
	return CommentItem{CommentID: "parent_1", MomentID: "m1", Content: "parent"}, true, nil
}

func coverStaticAppConfig(t *testing.T, mutate func(*appconfig.AppConfig)) appconfig.Source {
	t.Helper()
	src := appconfig.NewStaticSource(config.Config{})
	if _, err := src.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		mutate(&cfg)
		return cfg
	}); err != nil {
		t.Fatalf("update static app config: %v", err)
	}
	return src
}

func TestServiceGetDelegatesToRepository(t *testing.T) {
	repo := &stubCommentRepo{getResult: CommentItem{CommentID: "c1", Content: "hi"}, getFound: true}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	item, found, err := svc.Get(context.Background(), "c1")
	if err != nil || !found || item.CommentID != "c1" || item.Content != "hi" {
		t.Fatalf("unexpected get result: %#v found=%v err=%v", item, found, err)
	}

	repo.getFound = false
	if _, found, err := svc.Get(context.Background(), "missing"); err != nil || found {
		t.Fatalf("missing comment must be (false,nil), got found=%v err=%v", found, err)
	}

	repo.getErr = errors.New("get down")
	if _, _, err := svc.Get(context.Background(), "c1"); err == nil {
		t.Fatal("repository error must be surfaced")
	}
}

func TestServiceListFillsReactionsForItemsAndTopReplies(t *testing.T) {
	repo := &coverTopRepliesRepo{
		stubCommentRepo: &stubCommentRepo{
			listResult: PaginatedComments{Items: []CommentItem{
				{CommentID: "c1", RepliesCount: 1},
				{CommentID: "c2"},
			}},
		},
		result: map[string][]CommentItem{"c1": {{CommentID: "r1"}}},
	}
	aggregator := &coverReactionAggregator{result: map[string][]Reaction{
		"c1": {{Emoji: "👍", Count: 2, ReactedByMe: true}},
		"r1": {{Emoji: "❤️", Count: 1}},
	}}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	svc.SetReactionAggregator(aggregator)

	page, err := svc.List(testCommentContext("viewer_1"), ListCommentsRequest{MomentID: "moment_1"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if aggregator.calls != 1 {
		t.Fatalf("expected exactly one aggregate call, got %d", aggregator.calls)
	}
	if aggregator.viewerID != "viewer_1" {
		t.Fatalf("aggregator viewer = %q, want viewer_1", aggregator.viewerID)
	}
	sort.Strings(aggregator.ids)
	if strings.Join(aggregator.ids, ",") != "c1,c2,r1" {
		t.Fatalf("aggregate ids = %v, want [c1 c2 r1]", aggregator.ids)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %#v", page.Items)
	}
	if len(page.Items[0].Reactions) != 1 || page.Items[0].Reactions[0].Emoji != "👍" || !page.Items[0].Reactions[0].ReactedByMe {
		t.Fatalf("top-level reactions not filled: %#v", page.Items[0].Reactions)
	}
	if len(page.Items[1].Reactions) != 0 {
		t.Fatalf("comment without reactions must stay empty: %#v", page.Items[1].Reactions)
	}
	if len(page.Items[0].TopReplies) != 1 || page.Items[0].TopReplies[0].CommentID != "r1" {
		t.Fatalf("expected top reply r1, got %#v", page.Items[0].TopReplies)
	}
	if len(page.Items[0].TopReplies[0].Reactions) != 1 || page.Items[0].TopReplies[0].Reactions[0].Emoji != "❤️" {
		t.Fatalf("top reply reactions not filled: %#v", page.Items[0].TopReplies[0].Reactions)
	}

	// 匿名调用：viewer 为空串（reacted_by_me 恒 false 的聚合口径）。
	page, err = svc.List(context.Background(), ListCommentsRequest{MomentID: "moment_1"})
	if err != nil {
		t.Fatalf("anonymous list: %v", err)
	}
	if aggregator.calls != 2 || aggregator.viewerID != "" {
		t.Fatalf("anonymous aggregate calls=%d viewer=%q", aggregator.calls, aggregator.viewerID)
	}
	_ = page

	// 聚合失败：仅记 WARN，列表仍返回且 reactions 保持为空。
	// listResult 的切片会被成功路径原地填充，这里重置为未填充数据。
	repo.stubCommentRepo.listResult = PaginatedComments{Items: []CommentItem{
		{CommentID: "c1", RepliesCount: 1},
		{CommentID: "c2"},
	}}
	repo.result = map[string][]CommentItem{"c1": {{CommentID: "r1"}}}
	aggregator.err = errors.New("aggregate down")
	page, err = svc.List(testCommentContext("viewer_1"), ListCommentsRequest{MomentID: "moment_1"})
	if err != nil {
		t.Fatalf("aggregate failure must not fail the list: %v", err)
	}
	if len(page.Items[0].Reactions) != 0 || len(page.Items[0].TopReplies[0].Reactions) != 0 {
		t.Fatalf("reactions must stay empty on aggregate failure: %#v", page.Items[0])
	}

	// 空列表：不触发聚合查询。
	aggregator.err = nil
	aggregator.calls = 0
	repo.stubCommentRepo.listResult = PaginatedComments{Items: []CommentItem{}}
	if _, err := svc.List(testCommentContext("viewer_1"), ListCommentsRequest{MomentID: "moment_1"}); err != nil {
		t.Fatalf("empty list: %v", err)
	}
	if aggregator.calls != 0 {
		t.Fatalf("empty list must not aggregate, got %d calls", aggregator.calls)
	}
}

func TestServiceListTopRepliesConfigAndError(t *testing.T) {
	repo := &coverTopRepliesRepo{
		stubCommentRepo: &stubCommentRepo{
			listResult: PaginatedComments{Items: []CommentItem{{CommentID: "c1", RepliesCount: 2}}},
		},
		result: map[string][]CommentItem{"c1": {{CommentID: "r1"}, {CommentID: "r2"}}},
	}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)

	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.calls != 1 || repo.limit != 3 {
		t.Fatalf("default top replies limit must be 3, got calls=%d limit=%d", repo.calls, repo.limit)
	}
	if len(repo.parentIDs) != 1 || repo.parentIDs[0] != "c1" {
		t.Fatalf("unexpected parent ids: %#v", repo.parentIDs)
	}

	src := coverStaticAppConfig(t, func(cfg *appconfig.AppConfig) {
		cfg.ResponseDefaults.CommentTopRepliesCount = 1
	})
	svc = NewServiceWithDeps(repo, defaultMoments(), nil, nil, src, nil)
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err != nil {
		t.Fatalf("list with config: %v", err)
	}
	if repo.limit != 1 {
		t.Fatalf("configured top replies limit = %d, want 1", repo.limit)
	}

	// 非正配置回落默认 3。
	srcZero := coverStaticAppConfig(t, func(cfg *appconfig.AppConfig) {
		cfg.ResponseDefaults.CommentTopRepliesCount = 0
	})
	svc = NewServiceWithDeps(repo, defaultMoments(), nil, nil, srcZero, nil)
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err != nil {
		t.Fatalf("list with zero config: %v", err)
	}
	if repo.limit != 3 {
		t.Fatalf("zero config must fall back to 3, got %d", repo.limit)
	}

	// 批量查询失败：列表仍成功，top_replies 保持为空。
	// 前几次成功调用已把 top_replies 写回 listResult 的元素，这里重置。
	repo.stubCommentRepo.listResult = PaginatedComments{Items: []CommentItem{{CommentID: "c1", RepliesCount: 2}}}
	repo.err = errors.New("top replies down")
	page, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"})
	if err != nil {
		t.Fatalf("top replies failure must not fail the list: %v", err)
	}
	if len(page.Items[0].TopReplies) != 0 {
		t.Fatalf("top replies must stay empty on failure: %#v", page.Items[0].TopReplies)
	}

	// 无回复的列表不发起批量查询。
	repo.err = nil
	repo.calls = 0
	repo.stubCommentRepo.listResult = PaginatedComments{Items: []CommentItem{{CommentID: "c9"}}}
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err != nil {
		t.Fatalf("list without replies: %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("no replies must skip batch lookup, got %d calls", repo.calls)
	}
}

func TestServiceListPaginationAndErrors(t *testing.T) {
	repo := &coverSpyListRepo{stubCommentRepo: &stubCommentRepo{
		listResult: PaginatedComments{Items: []CommentItem{{CommentID: "c1"}}},
	}}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)

	page, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{
		MomentID:   "m1",
		Pagination: &PaginationParams{Limit: 500, Cursor: "cursor_1"},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.listLimit != types.MaxPageSize {
		t.Fatalf("limit must be clamped to %d, got %d", types.MaxPageSize, repo.listLimit)
	}
	if repo.listCursor != "cursor_1" || len(page.Items) != 1 {
		t.Fatalf("cursor must be passed through, got %q items=%d", repo.listCursor, len(page.Items))
	}

	// limit<=0 与 nil pagination 都回落默认 20。
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1", Pagination: &PaginationParams{}}); err != nil {
		t.Fatalf("list with empty pagination: %v", err)
	}
	if repo.listLimit != 20 || repo.listCursor != "" {
		t.Fatalf("default pagination = limit %d cursor %q", repo.listLimit, repo.listCursor)
	}
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err != nil {
		t.Fatalf("list without pagination: %v", err)
	}
	if repo.listLimit != 20 {
		t.Fatalf("nil pagination limit = %d, want 20", repo.listLimit)
	}

	repo.listErr = errors.New("list down")
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err == nil {
		t.Fatal("list repository error must be surfaced")
	}

	invisible := &stubMomentProvider{visibleErr: errors.New("visibility down")}
	svc = NewServiceWithDeps(repo, invisible, nil, nil, nil, nil)
	if _, err := svc.List(testCommentContext("viewer"), ListCommentsRequest{MomentID: "m1"}); err == nil {
		t.Fatal("visibility lookup error must be surfaced")
	}
}

func TestServiceListRepliesPaginationAndErrors(t *testing.T) {
	repo := &coverSpyListRepo{stubCommentRepo: &stubCommentRepo{
		getResult:  CommentItem{CommentID: "c1", MomentID: "m1"},
		getFound:   true,
		repliesRes: PaginatedComments{Items: []CommentItem{{CommentID: "r1"}}},
	}}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)

	page, err := svc.ListReplies(testCommentContext("viewer"), CommentRepliesRequest{
		CommentID:  "c1",
		Pagination: &PaginationParams{Limit: 500, Cursor: "reply_cursor"},
	})
	if err != nil {
		t.Fatalf("list replies: %v", err)
	}
	if repo.repliesLimit != types.MaxPageSize || repo.repliesCursor != "reply_cursor" || len(page.Items) != 1 {
		t.Fatalf("reply pagination = limit %d cursor %q items=%d", repo.repliesLimit, repo.repliesCursor, len(page.Items))
	}

	if _, err := svc.ListReplies(testCommentContext("viewer"), CommentRepliesRequest{CommentID: "c1"}); err != nil {
		t.Fatalf("list replies without pagination: %v", err)
	}
	if repo.repliesLimit != 20 || repo.repliesCursor != "" {
		t.Fatalf("default reply pagination = limit %d cursor %q", repo.repliesLimit, repo.repliesCursor)
	}

	repo.getErr = errors.New("get down")
	if _, err := svc.ListReplies(testCommentContext("viewer"), CommentRepliesRequest{CommentID: "c1"}); err == nil {
		t.Fatal("parent lookup error must be surfaced")
	}
	repo.getErr = nil

	invisible := &stubMomentProvider{visibleErr: errors.New("visibility down")}
	svc = NewServiceWithDeps(repo, invisible, nil, nil, nil, nil)
	if _, err := svc.ListReplies(testCommentContext("viewer"), CommentRepliesRequest{CommentID: "c1"}); err == nil {
		t.Fatal("visibility lookup error must be surfaced")
	}

	repo.getFound = true
	svc = NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	repo.repliesErr = errors.New("replies down")
	if _, err := svc.ListReplies(testCommentContext("viewer"), CommentRepliesRequest{CommentID: "c1"}); err == nil {
		t.Fatal("reply listing error must be surfaced")
	}
}

func TestServiceGetForDetailVisibilityAndReactions(t *testing.T) {
	repo := &stubCommentRepo{getResult: CommentItem{CommentID: "c1", MomentID: "m1"}, getFound: true}
	invisible := &stubMomentProvider{visibleErr: errors.New("visibility down")}
	svc := NewServiceWithDeps(repo, invisible, nil, nil, nil, nil)
	item, found, err := svc.GetForDetail(context.Background(), "c1")
	if err != nil || found || item.CommentID != "" {
		t.Fatalf("visibility failure must degrade to not-found, got %#v found=%v err=%v", item, found, err)
	}

	aggregator := &coverReactionAggregator{result: map[string][]Reaction{
		"c1": {{Emoji: "😂", Count: 3}},
	}}
	svc = NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	svc.SetReactionAggregator(aggregator)
	item, found, err = svc.GetForDetail(testCommentContext("viewer_2"), "c1")
	if err != nil || !found {
		t.Fatalf("get detail: found=%v err=%v", found, err)
	}
	if item.CommentID != "c1" {
		t.Fatalf("unexpected detail item: %#v", item)
	}
	// 详情读路径必须把 viewer 传给聚合器（viewer 决定 reacted_by_me 口径），
	// 并把聚合结果回写到返回 item（回归：曾因作用在临时切片上导致 reactions 恒空）。
	if aggregator.calls != 1 || aggregator.viewerID != "viewer_2" {
		t.Fatalf("aggregator calls=%d viewer=%q", aggregator.calls, aggregator.viewerID)
	}
	if len(aggregator.ids) != 1 || aggregator.ids[0] != "c1" {
		t.Fatalf("aggregator ids = %v, want [c1]", aggregator.ids)
	}
	if len(item.Reactions) != 1 || item.Reactions[0].Emoji != "😂" || item.Reactions[0].Count != 3 {
		t.Fatalf("detail reactions not filled: %#v", item.Reactions)
	}
}

func TestServiceCreateReplyFlattensNestedParent(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	ctx := testCommentContext("user_alice")

	parent, err := svc.Create(ctx, CreateCommentRequest{MomentID: "m1", Content: "parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	reply, err := svc.Create(ctx, CreateCommentRequest{MomentID: "m1", ParentID: parent.CommentID, Content: "reply"})
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	nested, err := svc.Create(ctx, CreateCommentRequest{MomentID: "m1", ParentID: reply.CommentID, Content: "nested"})
	if err != nil {
		t.Fatalf("create nested reply: %v", err)
	}
	if nested.ParentID == nil || *nested.ParentID != parent.CommentID {
		t.Fatalf("reply-to-reply must be flattened to the top-level parent, got %#v", nested.ParentID)
	}

	if _, err := svc.Create(ctx, CreateCommentRequest{MomentID: "m2", ParentID: parent.CommentID, Content: "cross moment"}); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("cross-moment parent must be rejected with ErrCommentNotFound, got %v", err)
	}
}

func TestServiceCreateMentionsLimit(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "user_alice"}, Content: "hi"}}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)

	tooMany := make([]string, mentionsMaxCount+1)
	for i := range tooMany {
		tooMany[i] = "user_" + strconv.Itoa(i)
	}
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi", Mentions: tooMany}); !errors.Is(err, ErrTooManyMentions) {
		t.Fatalf("expected ErrTooManyMentions, got %v", err)
	}

	boundary := tooMany[:mentionsMaxCount]
	item, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi", Mentions: boundary})
	if err != nil {
		t.Fatalf("mentions at the limit must be accepted, got %v", err)
	}
	if item.CommentID != "c1" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestServiceCreateTxRunnerBranches(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "user_alice"}, Content: "hi"}}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)

	txCalls := 0
	svc.runInTx = func(ctx context.Context, fn func(context.Context) error) error {
		txCalls++
		return fn(ctx)
	}
	item, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if err != nil || txCalls != 1 || item.CommentID != "c1" {
		t.Fatalf("transactional create: item=%#v calls=%d err=%v", item, txCalls, err)
	}

	svc.runInTx = func(context.Context, func(context.Context) error) error {
		return errors.New("tx failed")
	}
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err == nil {
		t.Fatal("transaction failure must be surfaced")
	}

	svc.runInTx = func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
	repo.createErr = errors.New("create failed")
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err == nil {
		t.Fatal("repository create failure inside tx must be surfaced")
	}
}

func TestServiceCreateNotificationContextFallbacks(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "user_alice"}, Content: "hi"}}
	moments := defaultMoments()

	moments.notifErr = errors.New("notification context down")
	svc := NewServiceWithDeps(repo, moments, nil, nil, nil, nil)
	item, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if err != nil || item.CommentID != "c1" {
		t.Fatalf("notification context failure must not fail create: item=%#v err=%v", item, err)
	}

	moments.notifErr = nil
	moments.notifOk = false
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err != nil {
		t.Fatalf("missing notification context must not fail create: %v", err)
	}

	moments.notifOk = true
	moments.notifCtx = MomentNotificationContext{}
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err != nil {
		t.Fatalf("empty moment author must not fail create: %v", err)
	}
}

func TestServiceCreateBlockCheckBranches(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "user_bob"}, Content: "hi"}}
	moments := defaultMoments()
	svc := NewServiceWithDeps(repo, moments, nil, nil, nil, nil)

	blockCalls := 0
	svc.SetBlockChecker(func(callerID, authorID string) (bool, bool, error) {
		blockCalls++
		if callerID != "user_bob" || authorID != "author1" {
			t.Fatalf("unexpected block check args: caller=%q author=%q", callerID, authorID)
		}
		return false, false, nil
	})

	// 作者未知：不查黑名单。
	moments.notifCtx = MomentNotificationContext{}
	moments.notifOk = true
	if _, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err != nil {
		t.Fatalf("create without author: %v", err)
	}
	if blockCalls != 0 {
		t.Fatalf("block check must be skipped when author is unknown, got %d calls", blockCalls)
	}

	// 作者已知：执行双向拉黑判定。
	moments.notifCtx = MomentNotificationContext{UserID: "author1"}
	if _, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err != nil {
		t.Fatalf("create with author: %v", err)
	}
	if blockCalls != 1 {
		t.Fatalf("expected one block check, got %d", blockCalls)
	}

	// 黑名单查询失败 → fail-closed 拒绝。
	svc.SetBlockChecker(func(string, string) (bool, bool, error) {
		return false, false, errors.New("block lookup failed")
	})
	if _, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err == nil {
		t.Fatal("block check failure must be surfaced (fail-closed)")
	}

	// 调用方缺失（匿名上下文）：跳过拉黑判定。
	blockCalls = 0
	svc.SetBlockChecker(func(string, string) (bool, bool, error) {
		blockCalls++
		return false, false, nil
	})
	if _, err := svc.Create(context.Background(), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err != nil {
		t.Fatalf("anonymous create: %v", err)
	}
	if blockCalls != 0 {
		t.Fatalf("block check must be skipped without a caller, got %d calls", blockCalls)
	}
}

func TestServiceCreateUsesAppConfigCommentMaxLength(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "user_alice"}, Content: "hi"}}
	src := coverStaticAppConfig(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.CommentMaxLength = 4
	})
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, src, nil)
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "abcde"}); !errors.Is(err, ErrCommentTooLong) {
		t.Fatalf("expected ErrCommentTooLong for 5 runes with max=4, got %v", err)
	}
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: "abcd"}); err != nil {
		t.Fatalf("content at configured limit must pass: %v", err)
	}

	// 非正配置回落默认 500。
	srcZero := coverStaticAppConfig(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.CommentMaxLength = 0
	})
	svc = NewServiceWithDeps(repo, defaultMoments(), nil, nil, srcZero, nil)
	if _, err := svc.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "m1", Content: strings.Repeat("评", defaultCommentMaxRunes+1)}); !errors.Is(err, ErrCommentTooLong) {
		t.Fatalf("expected default max length fallback, got %v", err)
	}
}

func TestServiceCreateReplyWithoutAuthorSkipsNotification(t *testing.T) {
	repo := replyWithoutAuthorRepo{stubCommentRepo: &stubCommentRepo{
		created: CommentItem{CommentID: "c2", MomentID: "m1", User: UserInfo{UserID: "user_bob"}, Content: "reply"},
	}}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	item, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "m1", ParentID: "parent_1", Content: "reply"})
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if item.CommentID != "c2" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestServiceDeleteRepositoryErrorsAndSoftDeleteTx(t *testing.T) {
	moments := &momentProviderStub{commentableFound: true, commentable: true}
	repo := &stubCommentRepo{getErr: errors.New("get down")}
	svc := NewServiceWithDeps(repo, moments, nil, nil, nil, nil)
	if err := svc.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: "c1"}); err == nil {
		t.Fatal("delete lookup error must be surfaced")
	}
	if err := svc.DeleteAsAdmin(testCommentContext("admin"), "c1"); err == nil {
		t.Fatal("admin delete lookup error must be surfaced")
	}

	repo = &stubCommentRepo{
		getResult: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "user_alice"}},
		getFound:  true,
	}
	svc = NewServiceWithDeps(repo, moments, nil, nil, nil, nil)
	txCalls := 0
	svc.runInTx = func(ctx context.Context, fn func(context.Context) error) error {
		txCalls++
		return fn(ctx)
	}
	if err := svc.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: "c1"}); err != nil {
		t.Fatalf("transactional delete: %v", err)
	}
	if txCalls != 1 {
		t.Fatalf("expected one transaction, got %d", txCalls)
	}
	if moments.increment != -1 {
		t.Fatalf("soft delete must decrement interaction count, got %d", moments.increment)
	}

	repo.deleteErr = errors.New("soft delete down")
	if err := svc.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: "c1"}); err == nil {
		t.Fatal("soft delete error must be surfaced")
	}

	repo.deleteErr = nil
	svc.runInTx = func(context.Context, func(context.Context) error) error {
		return errors.New("tx failed")
	}
	if err := svc.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: "c1"}); err == nil {
		t.Fatal("transaction failure must be surfaced")
	}
}

func TestBuildCommentActorFallbacks(t *testing.T) {
	actor := buildCommentActor(CommentItem{}, "fallback_user")
	if actor.UserID != "fallback_user" || actor.Nickname != "fallback_user" {
		t.Fatalf("empty user must fall back to caller id: %#v", actor)
	}
	if actor.AvatarURL != nil {
		t.Fatalf("missing avatar must stay nil: %#v", actor.AvatarURL)
	}

	actor = buildCommentActor(CommentItem{User: UserInfo{
		UserID:    " user_1 ",
		Nickname:  " Alice ",
		AvatarURL: strPtr(" https://cdn.example/a.png "),
	}}, "ignored")
	if actor.UserID != "user_1" || actor.Nickname != "Alice" {
		t.Fatalf("actor fields must be trimmed: %#v", actor)
	}
	if actor.AvatarURL == nil || *actor.AvatarURL != "https://cdn.example/a.png" {
		t.Fatalf("avatar must be trimmed: %#v", actor.AvatarURL)
	}
}

func TestUniqueMentionTargetsAndSummarizeCommentText(t *testing.T) {
	targets := uniqueMentionTargets([]string{" user_1 ", "", "   ", "user_2", "user_2", "self_user"}, "self_user")
	if len(targets) != 2 {
		t.Fatalf("expected two unique targets, got %#v", targets)
	}
	if _, ok := targets["user_1"]; !ok {
		t.Fatalf("whitespace mention must be trimmed: %#v", targets)
	}
	if _, ok := targets["self_user"]; ok {
		t.Fatalf("self mention must be dropped: %#v", targets)
	}

	if got := summarizeCommentText("", "  fallback text  "); got != "fallback text" {
		t.Fatalf("empty primary must fall back and trim, got %q", got)
	}
	long := strings.Repeat("评", 120)
	summary := summarizeCommentText(long, "ignored")
	if runes := []rune(summary); len(runes) != 80 {
		t.Fatalf("summary must truncate to 80 runes, got %d", len(runes))
	}
	if !strings.HasPrefix(long, summary) {
		t.Fatalf("summary must be a rune-prefix of the original, got %q", summary)
	}
}

// TestServiceCreatePropagatesMomentCommentableError 覆盖 ensureMomentCommentable
// 对 CheckCommentable 查询失败的透传（不可判定可评论性时拒绝写入）。
func TestServiceCreatePropagatesMomentCommentableError(t *testing.T) {
	moments := &stubMomentProvider{commentableErr: errors.New("commentable down")}
	svc := NewServiceWithDeps(&stubCommentRepo{}, moments, nil, nil, nil, nil)
	if _, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "m1", Content: "hi"}); err == nil {
		t.Fatal("commentable lookup error must be surfaced")
	}
}

// TestServiceCreatePropagatesParentLookupError 覆盖 resolveReplyTarget 的
// 仓储错误透传（父评论查询失败不能降级为「无父评论」）。
func TestServiceCreatePropagatesParentLookupError(t *testing.T) {
	repo := &stubCommentRepo{getErr: errors.New("parent lookup down")}
	svc := NewServiceWithDeps(repo, defaultMoments(), nil, nil, nil, nil)
	if _, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "m1", ParentID: "comment_parent", Content: "reply"}); err == nil {
		t.Fatal("parent lookup error must be surfaced")
	}
}

// TestServiceCreateReplyIgnoresNotificationFailure 覆盖回复通知写失败时
// 评论仍创建成功的降级口径（AppendCommentReply / AppendMentionedBatch 同批失败）。
func TestServiceCreateReplyIgnoresNotificationFailure(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		commentableFound: true,
		commentable:      true,
		notificationOK:   true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "m1"},
		},
	}
	writer := &failingNotificationWriter{}
	svc := NewServiceWithDeps(repo, moments, writer, nil, nil, nil)
	parent, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "m1", Content: "parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	reply, err := svc.Create(testCommentContext("user_bob"), CreateCommentRequest{
		MomentID: "m1",
		ParentID: parent.CommentID,
		Content:  "reply",
		Mentions: []string{"user_carol"},
	})
	if err != nil {
		t.Fatalf("notification failure must not fail create: %v", err)
	}
	if reply.CommentID == "" {
		t.Fatalf("expected created reply, got %#v", reply)
	}
	if writer.calls != 2 {
		t.Fatalf("expected reply + mention notification attempts, got %d", writer.calls)
	}
}
