package notify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/go-chi/chi/v5"
)

// batchErrStub 让 AppendBatchPerUser 失败的桩仓库，用于验证批量 append
// 失败整体报错（调用方按通知缺失降级推送）。
type batchErrStub struct {
	Repository
	batchErr error
}

func (s *batchErrStub) AppendBatchPerUser(context.Context, []string, []NotifyItem) error {
	return s.batchErr
}

// pruneRepoErrStub 让 PruneReadBefore 返回错误的桩仓库（其余行为同内存实现），
// 用于验证 sweeper 失败只记日志、不 panic。
type pruneRepoErrStub struct {
	Repository
	pruneErr error
}

func (s *pruneRepoErrStub) PruneReadBefore(context.Context, time.Time) (int64, error) {
	return 0, s.pruneErr
}

func TestNewAnnouncementNotifyIDGeneratesUniqueIDs(t *testing.T) {
	service := NewService()
	first := service.NewAnnouncementNotifyID()
	second := service.NewAnnouncementNotifyID()
	if first == "" || second == "" || first == second {
		t.Fatalf("expected unique non-empty ids, got %q / %q", first, second)
	}
}

func TestAppendMentionedBatchFansOutPerUser(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ids, err := service.AppendMentionedBatch(context.Background(),
		[]string{"user_alice", "user_carol"},
		types.UserInfo{UserID: "user_bob", Nickname: "Bob"},
		"moment", "moment_1",
		types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"},
		"comment_1", "preview")
	if err != nil {
		t.Fatalf("append mentioned batch: %v", err)
	}
	if len(ids) != 2 || ids["user_alice"] == "" || ids["user_carol"] == "" || ids["user_alice"] == ids["user_carol"] {
		t.Fatalf("expected per-user independent ids, got %#v", ids)
	}
	for _, userID := range []string{"user_alice", "user_carol"} {
		page, err := repo.List(context.Background(), userID, NotifyListRequest{})
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("expected 1 notify for %s, got %#v err=%v", userID, page, err)
		}
		item := page.Items[0]
		if item.NotifyID != ids[userID] || item.Type != "mentioned" || item.IsRead {
			t.Fatalf("unexpected item for %s: %#v", userID, item)
		}
		if item.Payload["comment_id"] != "comment_1" || item.Payload["comment_preview"] != "preview" {
			t.Fatalf("expected comment fields in payload: %#v", item.Payload)
		}
	}
}

func TestAppendMentionedBatchOmitsEmptyCommentFields(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ids, err := service.AppendMentionedBatch(context.Background(),
		[]string{"user_alice"},
		types.UserInfo{UserID: "user_bob", Nickname: "Bob"},
		"moment", "moment_1",
		types.MomentSummary{MomentID: "moment_1"}, "", "")
	if err != nil {
		t.Fatalf("append mentioned batch: %v", err)
	}
	page, err := repo.List(context.Background(), "user_alice", NotifyListRequest{})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("expected 1 notify, got %#v err=%v", page, err)
	}
	payload := page.Items[0].Payload
	if _, ok := payload["comment_id"]; ok {
		t.Fatalf("comment_id should be omitted when empty: %#v", payload)
	}
	if _, ok := payload["comment_preview"]; ok {
		t.Fatalf("comment_preview should be omitted when empty: %#v", payload)
	}
	if ids["user_alice"] == "" {
		t.Fatal("expected returned id mapping to contain user_alice")
	}
}

func TestAppendMentionedBatchPropagatesRepoError(t *testing.T) {
	service := NewServiceWithRepo(&batchErrStub{batchErr: errors.New("batch down")})
	if _, err := service.AppendMentionedBatch(context.Background(), []string{"user_alice"},
		types.UserInfo{UserID: "user_bob"}, "moment", "moment_1", types.MomentSummary{}, "", ""); err == nil {
		t.Fatal("expected batch append error, got nil")
	}
}

func TestAppendGroupMentionedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	id, err := service.AppendGroupMentioned(context.Background(), "user_alice",
		types.UserInfo{UserID: "user_bob", Nickname: "Bob"},
		"grp_1", "测试群", "gm_1", "群聊预览")
	if err != nil {
		t.Fatalf("append group mentioned: %v", err)
	}
	item := repo.appended[0]
	if id == "" || repo.appendUser != "user_alice" {
		t.Fatalf("unexpected append: id=%q user=%q", id, repo.appendUser)
	}
	if item.Type != "group_mentioned" || item.Payload["group_id"] != "grp_1" ||
		item.Payload["group_name"] != "测试群" || item.Payload["message_id"] != "gm_1" ||
		item.Payload["content_preview"] != "群聊预览" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestAppendGroupMentionedOmitsEmptyPreview(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendGroupMentioned(context.Background(), "user_alice",
		types.UserInfo{UserID: "user_bob"}, "grp_1", "测试群", "gm_1", ""); err != nil {
		t.Fatalf("append group mentioned: %v", err)
	}
	page, err := repo.List(context.Background(), "user_alice", NotifyListRequest{})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("expected 1 notify, got %#v err=%v", page, err)
	}
	if _, ok := page.Items[0].Payload["content_preview"]; ok {
		t.Fatalf("content_preview should be omitted when empty: %#v", page.Items[0].Payload)
	}
}

func TestAppendGroupMentionedBatchFansOutPerUser(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ids, err := service.AppendGroupMentionedBatch(context.Background(),
		[]string{"user_alice", "user_carol"},
		types.UserInfo{UserID: "user_bob"}, "grp_1", "测试群", "gm_1", "预览")
	if err != nil {
		t.Fatalf("append group mentioned batch: %v", err)
	}
	if len(ids) != 2 || ids["user_alice"] == ids["user_carol"] {
		t.Fatalf("expected per-user independent ids, got %#v", ids)
	}
	for _, userID := range []string{"user_alice", "user_carol"} {
		page, err := repo.List(context.Background(), userID, NotifyListRequest{})
		if err != nil || len(page.Items) != 1 || page.Items[0].NotifyID != ids[userID] {
			t.Fatalf("unexpected item for %s: %#v err=%v", userID, page, err)
		}
	}
}

func TestAppendGroupMentionedBatchPropagatesRepoError(t *testing.T) {
	service := NewServiceWithRepo(&batchErrStub{batchErr: errors.New("batch down")})
	if _, err := service.AppendGroupMentionedBatch(context.Background(), []string{"user_alice"},
		types.UserInfo{UserID: "user_bob"}, "grp_1", "测试群", "gm_1", ""); err == nil {
		t.Fatal("expected batch append error, got nil")
	}
}

func TestPruneOnceDeletesOnlyExpiredRead(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	now := time.Now().UTC()
	expired := now.Add(-notifyPruneRetention - 24*time.Hour).Format(time.RFC3339)
	fresh := now.Format(time.RFC3339)
	// 已读 + 过期 → 删；未读 + 过期 → 留；已读 + 新鲜 → 留；非法时间戳 → 留
	for _, item := range []NotifyItem{
		{NotifyID: "n_expired_read", Type: "system", IsRead: true, CreatedAt: expired},
		{NotifyID: "n_expired_unread", Type: "system", IsRead: false, CreatedAt: expired},
		{NotifyID: "n_fresh_read", Type: "system", IsRead: true, CreatedAt: fresh},
		{NotifyID: "n_bad_time", Type: "system", IsRead: true, CreatedAt: "not-rfc3339"},
	} {
		if err := repo.Append(context.Background(), "user_alice", item); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	service.pruneOnce(now)
	page, err := repo.List(context.Background(), "user_alice", NotifyListRequest{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := make(map[string]bool, len(page.Items))
	for _, item := range page.Items {
		got[item.NotifyID] = true
	}
	if got["n_expired_read"] {
		t.Fatal("expired read notify should be pruned")
	}
	for _, kept := range []string{"n_expired_unread", "n_fresh_read", "n_bad_time"} {
		if !got[kept] {
			t.Fatalf("notify %s should be kept", kept)
		}
	}
}

func TestPruneOnceSwallowsRepoError(t *testing.T) {
	service := NewServiceWithRepo(&pruneRepoErrStub{pruneErr: errors.New("db down")})
	service.pruneOnce(time.Now()) // 只记日志，不得 panic
}

func TestStartStopPruneSweeperLifecycle(t *testing.T) {
	StopPruneSweeper() // 确保包级状态从干净态开始
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := NewService()
	service.StartPruneSweeper(ctx)
	service.StartPruneSweeper(ctx) // 重复启动为 no-op
	time.Sleep(20 * time.Millisecond)
	StopPruneSweeper()
	time.Sleep(20 * time.Millisecond) // 给 sweeper goroutine 退出机会
	StopPruneSweeper()                // 幂等
	// 重启后可再次工作
	service.StartPruneSweeper(ctx)
	time.Sleep(20 * time.Millisecond)
	StopPruneSweeper()
}

func TestNotifyModuleWiring(t *testing.T) {
	fac := persistence.NewFactory(config.Config{})
	mod := NewModule(fac)
	if mod == nil || mod.handler == nil {
		t.Fatal("expected non-nil notify module")
	}
	r := chi.NewRouter()
	mod.Mount(r)

	// NewModuleWithService 同样可装配，且路由注册生效：未带 token 请求被
	// RequiredAuth 拦截（契约：HTTP 200 + 业务码 10100），证明路由组与
	// 中间件链已挂上。
	svcMod := NewModuleWithService(NewService())
	svcRouter := chi.NewRouter()
	svcMod.Mount(svcRouter)
	rec := httptest.NewRecorder()
	svcRouter.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/notify/fetch", strings.NewReader("{}")))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), strconv.Itoa(int(httpx.CodeUnauthorized))) {
		t.Fatalf("expected business code 10100 without token, got http=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 带 principal 的端到端小循环：mount 后用 WithPrincipal 无法绕过真实鉴权中间件，
// 这里退而验证 service 层 List 的 principal 读取路径（handler 已有覆盖）。
func TestServiceListReadsPrincipal(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	if err := repo.Append(context.Background(), "user_alice", NotifyItem{NotifyID: "n_1", Type: "system", IsRead: false}); err != nil {
		t.Fatalf("append: %v", err)
	}
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	page, err := service.List(ctx, NotifyListRequest{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].NotifyID != "n_1" {
		t.Fatalf("unexpected page: %#v", page)
	}
}
