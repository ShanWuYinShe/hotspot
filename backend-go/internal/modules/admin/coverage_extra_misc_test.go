package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// 本文件覆盖装配辅助（module/tagBlockChecker）、游标编解码的非法输入、
// 审核参数校验、事务跑批分支、审计元数据组装与持久化审核存储的错误出口。

func TestNewModuleWiring(t *testing.T) {
	// NewModule 只做装配，不触库：nil source 也不应在构造期 panic。
	module := NewModule(nil, nil, nil, nil, nil)
	if module == nil || module.handler == nil || module.Service == nil {
		t.Fatalf("module not wired: %#v", module)
	}
}

func TestTagBlockChecker(t *testing.T) {
	// nil module / 未装配 handler 时必须安全返回 false。
	var nilChecker tagBlockChecker
	if nilChecker.IsBlocked("hotspot") {
		t.Fatal("nil checker must report unblocked")
	}
	if (&Module{}).BlockedTagChecker().IsBlocked("hotspot") {
		t.Fatal("module without handler must report unblocked")
	}

	module, _ := newTestModuleWithStore()
	store, ok := module.Service.moderationStore.(*InMemoryModerationStore)
	if !ok {
		t.Fatalf("unexpected moderation store type: %T", module.Service.moderationStore)
	}
	checker := module.BlockedTagChecker()
	if checker.IsBlocked("hotspot") {
		t.Fatal("tag should not be blocked yet")
	}
	if err := store.SetTagBan(context.Background(), "hotspot", true, false); err != nil {
		t.Fatalf("block tag: %v", err)
	}
	if !checker.IsBlocked("hotspot") {
		t.Fatal("blocked tag should be reported blocked")
	}
}

func TestCursorEncodeDecodeInvalidBranches(t *testing.T) {
	// json.Marshal 失败（chan 不可序列化）→ 编码返回空串。
	if got := encodeCursor[chan int](adminCursorNamespace, make(chan int)); got != "" {
		t.Fatalf("encodeCursor(chan) = %q, want empty", got)
	}
	// 空 raw 直接返回零值，不视为错误。
	if c, apiErr := decodeCursor[adminCursor](adminCursorNamespace, "   "); apiErr != nil || c != (adminCursor{}) {
		t.Fatalf("decodeCursor empty = (%#v, %v)", c, apiErr)
	}

	// 编码后语义校验失败：各列表游标字段缺失/时间零值 → 10000。
	if _, apiErr := decodeAdminCursor(encodeAdminCursor(adminCursor{})); apiErr == nil {
		t.Fatal("decodeAdminCursor(zero) should fail")
	}
	if _, apiErr := decodeAuditCursor(encodeAuditCursor(auditCursor{})); apiErr == nil {
		t.Fatal("decodeAuditCursor(zero) should fail")
	}
	if _, apiErr := decodeAdminUserCursor(encodeAdminUserCursor(adminUserCursor{})); apiErr == nil {
		t.Fatal("decodeAdminUserCursor(zero) should fail")
	}
	if _, apiErr := decodeAdminGroupCursor(encodeAdminGroupCursor(adminGroupCursor{})); apiErr == nil {
		t.Fatal("decodeAdminGroupCursor(zero) should fail")
	}
	if _, apiErr := decodeAdminReportCursor(encodeAdminReportCursor(adminReportCursor{})); apiErr == nil {
		t.Fatal("decodeAdminReportCursor(zero) should fail")
	}
	if _, apiErr := decodeTagCursor(encodeTagCursor(tagCursor{})); apiErr == nil {
		t.Fatal("decodeTagCursor(zero) should fail")
	}
	if _, apiErr := decodeAnnouncementCursor(encodeAnnouncementCursor(announcementCursor{})); apiErr == nil {
		t.Fatal("decodeAnnouncementCursor(zero) should fail")
	}

	// 正常往返。
	now := time.Now().UTC().Truncate(time.Second)
	if _, apiErr := decodeAdminCursor(encodeAdminCursor(adminCursor{UserID: "user_alice", GrantedAt: now})); apiErr != nil {
		t.Fatalf("valid admin cursor rejected: %v", apiErr)
	}
	if got := normalizeCursorLimit(types.MaxPageSize + 5); got != types.MaxPageSize {
		t.Fatalf("normalizeCursorLimit = %d, want %d", got, types.MaxPageSize)
	}
	if got := normalizeCursorLimit(0); got != 20 {
		t.Fatalf("normalizeCursorLimit(0) = %d, want 20", got)
	}
}

func TestModerationValidationHelpers(t *testing.T) {
	// 公告创建校验：级别 / 标题 / 内容 / 链接。
	if apiErr := validateAnnouncementCreate(CreateAnnouncementRequest{Title: "t", Content: "c", Severity: "bogus"}); apiErr == nil {
		t.Fatal("invalid severity should fail")
	}
	if apiErr := validateAnnouncementCreate(CreateAnnouncementRequest{Title: "  ", Content: "c", Severity: AnnounceLevelInfo}); apiErr == nil {
		t.Fatal("blank title should fail")
	}
	if apiErr := validateAnnouncementCreate(CreateAnnouncementRequest{Title: "t", Content: " ", Severity: AnnounceLevelInfo}); apiErr == nil {
		t.Fatal("blank content should fail")
	}
	if apiErr := validateAnnouncementCreate(CreateAnnouncementRequest{Title: "t", Content: "c", Severity: AnnounceLevelInfo, ActionURL: "ftp://example.com/x"}); apiErr == nil {
		t.Fatal("invalid action url should fail")
	}

	// 公告更新校验。
	if apiErr := validateAnnouncementUpdate(UpdateAnnouncementRequest{}); apiErr == nil {
		t.Fatal("empty id should fail")
	}
	if apiErr := validateAnnouncementUpdate(UpdateAnnouncementRequest{AnnouncementID: "ann"}); apiErr == nil {
		t.Fatal("no changed fields should fail")
	}
	blank := " "
	invalidLevel := AnnounceSeverity("bogus")
	invalidURL := "javascript:alert(1)"
	if apiErr := validateAnnouncementUpdate(UpdateAnnouncementRequest{AnnouncementID: "ann", Title: &blank}); apiErr == nil {
		t.Fatal("blank title should fail")
	}
	if apiErr := validateAnnouncementUpdate(UpdateAnnouncementRequest{AnnouncementID: "ann", Content: &blank}); apiErr == nil {
		t.Fatal("blank content should fail")
	}
	if apiErr := validateAnnouncementUpdate(UpdateAnnouncementRequest{AnnouncementID: "ann", Severity: &invalidLevel}); apiErr == nil {
		t.Fatal("invalid severity should fail")
	}
	if apiErr := validateAnnouncementUpdate(UpdateAnnouncementRequest{AnnouncementID: "ann", ActionURL: &invalidURL}); apiErr == nil {
		t.Fatal("invalid action url should fail")
	}

	// 标签搜索关键词长度上限。
	if apiErr := validateTagSearchRequest(AdminTagSearchRequest{Keyword: strings.Repeat("x", keywordMaxLength+1)}); apiErr == nil {
		t.Fatal("overlong keyword should fail")
	}
	if !validAnnounceLevel(AnnounceLevelWarning) || validAnnounceLevel("bogus") {
		t.Fatal("validAnnounceLevel mismatch")
	}
	if validAnnouncementActionURL("mailto:a@b.c") {
		t.Fatal("mailto url should be rejected")
	}
	if !validAnnouncementActionURL("") {
		t.Fatal("empty url should be accepted")
	}
}

func TestRunDecisionInTxBranches(t *testing.T) {
	ctx := context.Background()
	svc := covPermsService(NewInMemoryRepository(), NewInMemoryModerationStore(), nil)

	// 业务错误通过哨兵触发事务回滚，但对外仍返回 APIError。
	svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		_ = fn(ctx)
		return errAdminDecisionAborted
	})
	apiErr, err := svc.runDecisionInTx(ctx, func(context.Context) (*httpx.APIError, error) {
		return httpx.InvalidRequest("参数错误"), nil
	})
	if err != nil || apiErr == nil {
		t.Fatalf("aborted decision: apiErr=%v err=%v", apiErr, err)
	}

	// 无业务错误但回调返回原始错误 → 上抛。
	svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		_ = fn(ctx)
		return nil
	})
	if _, err := svc.runDecisionInTx(ctx, func(context.Context) (*httpx.APIError, error) {
		return nil, errCovAdmin
	}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("op error: %v", err)
	}

	// 事务本身失败 → 上抛事务错误（非业务哨兵）。
	svc.SetTxRunner(func(context.Context, func(context.Context) error) error { return errCovAdmin })
	if _, err := svc.runDecisionInTx(ctx, func(context.Context) (*httpx.APIError, error) {
		return nil, nil
	}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("tx error: %v", err)
	}

	// nil runner 时直接执行。
	svc.SetTxRunner(nil)
	if apiErr, err := svc.runDecisionInTx(ctx, func(context.Context) (*httpx.APIError, error) {
		return nil, errCovAdmin
	}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("nil runner: apiErr=%v err=%v", apiErr, err)
	}
}

func TestNewServiceRequiresModerationStore(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewService without moderation store should panic")
		}
	}()
	NewService(NewInMemoryRepository(), &stubProvider{}, nil, nil)
}

func TestInvalidateBanCacheHook(t *testing.T) {
	svc := covPermsService(NewInMemoryRepository(), NewInMemoryModerationStore(), nil)
	called := ""
	svc.SetBanCacheInvalidator(func(userIDs ...string) {
		if len(userIDs) > 0 {
			called = userIDs[0]
		}
	})
	svc.invalidateBanCache("user_bob")
	if called != "user_bob" {
		t.Fatalf("invalidator not called: %q", called)
	}
}

func TestNewAuditEntryMetadataBranches(t *testing.T) {
	// actor 无昵称时回填 user_id。
	entry := newAuditEntry(context.Background(), AdminActor{UserID: "admin_no_nick"}, AuditLogEntry{Action: AuditActionRoleCreate})
	if entry.Operator.UserID != "admin_no_nick" || entry.Operator.Nickname != "admin_no_nick" {
		t.Fatalf("operator fallback failed: %#v", entry.Operator)
	}

	// 请求上下文：request id 生成 log_id，UA 超长截断，国家落 IPLocation。
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("User-Agent", strings.Repeat("a", 150))
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	req.Header.Set("CF-IPCountry", "JP")
	rctx := withRequest(context.Background(), req)
	reqEntry := newAuditEntry(rctx, AdminActor{UserID: "admin"}, AuditLogEntry{Action: AuditAction("user.ban"), IPAddress: "1.2.3.4"})
	if reqEntry.DeviceSummary == "" || len(reqEntry.DeviceSummary) > 120 {
		t.Fatalf("device summary not truncated: %q", reqEntry.DeviceSummary)
	}
	if reqEntry.IPLocation != "JP" {
		t.Fatalf("IP location = %q, want JP", reqEntry.IPLocation)
	}

	// 请求携带 X-Request-Id 时审计 log_id 带该 ID。
	captured := ""
	middleware := observability.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerReq := adminContext(r, "admin", true)
		svc := covPermsService(NewInMemoryRepository(), NewInMemoryModerationStore(), nil)
		_, apiErr, err := svc.CreateRole(withRequest(innerReq.Context(), innerReq), AdminRbacRoleCreateRequest{Name: "cov_req_id_role"})
		if apiErr != nil || err != nil {
			t.Errorf("create role: apiErr=%v err=%v", apiErr, err)
		}
		captured = observability.RequestIDFromContext(innerReq.Context())
	}))
	httpReq := httptest.NewRequest(http.MethodPost, "/admin/api/v1/rbac/roles/create", strings.NewReader("{}"))
	httpReq.Header.Set("X-Request-Id", "cov-request-id-1234")
	middleware.ServeHTTP(httptest.NewRecorder(), httpReq)
	if captured != "cov-request-id-1234" {
		t.Fatalf("request id not propagated: %q", captured)
	}

	// 空 UA 返回空摘要。
	if got := deviceSummaryFromContext(context.Background()); got != "" {
		t.Fatalf("deviceSummaryFromContext(background) = %q, want empty", got)
	}
	if got := deviceSummaryFromContext(withRequest(context.Background(), httptest.NewRequest(http.MethodPost, "/x", nil))); got != "" {
		t.Fatalf("empty UA summary = %q, want empty", got)
	}
}

func TestHandlerListPermissionsSuccess(t *testing.T) {
	handler := newTestHandler()
	rr := httptest.NewRecorder()
	handler.ListPermissions(rr, adminContext(httptest.NewRequest(http.MethodPost, "/permissions/fetch", strings.NewReader("{}")), "admin", true))
	resp := decodeResponse[AdminPermissionDictionary](t, rr)
	if resp.Code != 0 || len(resp.Data.Items) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestHandlerStatsDecodeErrorBranches(t *testing.T) {
	handler := newTestHandler()
	for _, tc := range []struct {
		name string
		call func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{name: "stats", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.StatsGeo(w, r) }},
		{name: "drilldown", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.StatsGeoDrilldown(w, r) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(handler, rr, adminContext(httptest.NewRequest(http.MethodPost, "/stats", strings.NewReader("{")), "admin", true))
			resp := decodeResponse[struct{}](t, rr)
			if resp.Code != httpx.CodeInvalidRequest {
				t.Fatalf("unexpected response: %#v", resp)
			}
		})
	}
}

func TestServiceListAuditLogsBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	// 非法时间窗 → 服务层参数错误。
	if _, apiErr, err := svc.ListAuditLogs(ctx, AuditLogListRequest{StartTime: "yesterday"}); err != nil || apiErr == nil {
		t.Fatalf("invalid filter: apiErr=%v err=%v", apiErr, err)
	}
	// 仓储 APIError / 原始错误。
	repo.listAuditLogsErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
	if _, apiErr, err := svc.ListAuditLogs(ctx, AuditLogListRequest{}); apiErr == nil || err != nil {
		t.Fatalf("repo api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.listAuditLogsErr = errCovAdmin
	if _, apiErr, err := svc.ListAuditLogs(ctx, AuditLogListRequest{}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("repo raw error: apiErr=%v err=%v", apiErr, err)
	}
	repo.listAuditLogsErr = nil

	// 有下一页时编码游标。
	for i := 0; i < 3; i++ {
		if err := repo.InMemoryRepository.AppendAuditLog(ctx, AuditLogEntry{Action: AuditAction("user.ban"), TargetID: "user_x", CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatalf("append audit: %v", err)
		}
	}
	resp, apiErr, err := svc.ListAuditLogs(ctx, AuditLogListRequest{Pagination: &PaginationParams{Limit: 1}})
	if apiErr != nil || err != nil {
		t.Fatalf("list audit logs: apiErr=%v err=%v", apiErr, err)
	}
	if resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("expected next cursor: %#v", resp)
	}
}

func TestServiceBanUserValidationBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	// 封禁参数非法（user_id 空 / 时长负数）。
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{Active: true, Reason: "spam"}); err != nil || apiErr == nil {
		t.Fatalf("empty ban user: apiErr=%v err=%v", apiErr, err)
	}
	negative := -1
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam", DurationDays: &negative}); err != nil || apiErr == nil {
		t.Fatalf("negative duration: apiErr=%v err=%v", apiErr, err)
	}
	// 封禁不存在的用户。
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_nobody", Active: true, Reason: "spam"}); err != nil || apiErr == nil {
		t.Fatalf("missing user: apiErr=%v err=%v", apiErr, err)
	}
	// 取目标管理员资料失败。
	repo.getActorErrUser = "user_bob"
	repo.getActorErr = errCovAdmin
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("target actor error: %v", err)
	}
	repo.getActorErr = nil
	repo.getActorErrUser = ""
	// 低权限管理员不能封禁管理员目标。
	base.banStates["admin"] = UserBanState{Status: UserAccountStatusNormal}
	covGrantPermissions(t, base, "user_alice", "cov_limited_ban2", permissionUsersWrite)
	if apiErr, err := svc.SetUserBan(adminCtx("user_alice"), AdminSetUserBanRequest{UserID: "admin", Active: true, Reason: "spam"}); err != nil || apiErr == nil {
		t.Fatalf("protected target: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceHandleReportBanPropagationErrors(t *testing.T) {
	ctx := adminCtx("admin")
	newSvc := func(t *testing.T) (*Service, *covRepo) {
		t.Helper()
		svc, repo := covNewReportService(NewInMemoryModerationStore(), nil)
		svc.SetReportAdminStore(testReportAdminStore{store: repo.InMemoryRepository})
		svc.SetAccountStateWriter(testAccountStateWriter{store: repo.InMemoryRepository})
		svc.SetSessionAdmin(testSessionAdmin{store: repo.InMemoryRepository})
		return svc, repo
	}
	negative := -1

	t.Run("group not found", func(t *testing.T) {
		svc, repo := newSvc(t)
		svc.SetGroupModerator(covGroupModerator{err: errGroupNotFoundForCov()})
		seedPendingReport(t, repo.InMemoryRepository, "report_group", "user_reporter", "group", "group_missing")
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_group", Action: ReportActionResolve})
		if err != nil || apiErr == nil {
			t.Fatalf("group not found: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("user target ban invalid duration", func(t *testing.T) {
		svc, repo := newSvc(t)
		seedPendingReport(t, repo.InMemoryRepository, "report_user_ban", "user_reporter", "user", "user_bob")
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_user_ban", Action: ReportActionResolve, BanDurationDays: &negative})
		if err != nil || apiErr == nil {
			t.Fatalf("user ban invalid: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("moment target ban invalid duration", func(t *testing.T) {
		svc, repo := newSvc(t)
		svc.SetMomentModerator(testMomentModerator{store: repo.InMemoryRepository})
		seedPendingReport(t, repo.InMemoryRepository, "report_moment_ban", "user_reporter", "moment", "moment_cov")
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_moment_ban", Action: ReportActionResolve, BanDurationDays: &negative})
		if err != nil || apiErr == nil {
			t.Fatalf("moment ban invalid: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("comment target ban invalid duration", func(t *testing.T) {
		svc, repo := newSvc(t)
		svc.SetCommentModerator(testCommentModerator{store: repo.InMemoryRepository})
		seedPendingReport(t, repo.InMemoryRepository, "report_comment_ban", "user_reporter", "comment", "comment_cov")
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_comment_ban", Action: ReportActionResolve, BanDurationDays: &negative})
		if err != nil || apiErr == nil {
			t.Fatalf("comment ban invalid: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("message target ban invalid duration", func(t *testing.T) {
		svc, repo := newSvc(t)
		svc.SetMessageModerators(testMessageModerator{store: repo.InMemoryRepository}, testMessageModerator{store: repo.InMemoryRepository})
		seedPendingReport(t, repo.InMemoryRepository, "report_msg_ban", "user_reporter", "message", "message_cov")
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_msg_ban", Action: ReportActionResolve, BanDurationDays: &negative})
		if err != nil || apiErr == nil {
			t.Fatalf("message ban invalid: apiErr=%v err=%v", apiErr, err)
		}
	})
}

// errGroupNotFoundForCov 返回 groups 模块的 not found 哨兵（避免重复 import 别名）。
func errGroupNotFoundForCov() error {
	_, err := testGroupModerator{store: NewInMemoryRepository()}.DismissAsAdmin(context.Background(), "admin", "missing", "")
	return err
}

func TestModerationPersistenceStoreErrorBranches(t *testing.T) {
	ctx := context.Background()

	// factory 不可用：标签 API 返回 ErrNotImplemented。
	emptyStore := NewPersistenceModerationStore(nil, nil, nil)
	if err := emptyStore.SetTagBan(ctx, "hotspot", false, false); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("SetTagBan without db = %v", err)
	}
	if _, _, err := emptyStore.GetTagDeleteState(ctx, "hotspot"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetTagDeleteState without db = %v", err)
	}

	// 可查询后端 + 未装配 moment tag store：GetTagDeleteState 显式失败。
	backend := &adminFullStub{queryRow: map[string][]any{queryBlockedTagExists: {true}}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	storeNoTags := NewPersistenceModerationStore(factory, nil, nil)
	if _, _, err := storeNoTags.GetTagDeleteState(ctx, "hotspot"); !errors.Is(err, errMomentTagStoreNotWired) {
		t.Fatalf("nil moment tag store = %v", err)
	}
	storeBeforeTags := NewPersistenceModerationStore(factory, nil, covMomentTagStore{})
	if _, _, err := storeBeforeTags.GetTagDeleteState(ctx, "hotspot"); !errors.Is(err, errCovAdmin) {
		t.Fatalf("moment tag store error = %v", err)
	}

	// 解屏蔽 Exec 失败。
	failingExec := &adminFullStub{execErrs: map[string]error{queryUnblockTag: errCovAdmin}, queryRow: map[string][]any{queryBlockedTagExists: {true}}}
	factoryExec := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factoryExec.DB.(*persistence.PostgresDB).Backend = failingExec
	storeExec := NewPersistenceModerationStore(factoryExec, nil, nil)
	if err := storeExec.SetTagBan(ctx, "hotspot", false, false); !errors.Is(err, errCovAdmin) {
		t.Fatalf("unblock exec error = %v", err)
	}

	// SearchTags rows.Err 失败。
	rowsErrBackend := &adminFullStub{queryCustomRows: map[string]persistence.Rows{querySearchTags: &adminRowsValues{err: errCovAdmin}}}
	factoryRows := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factoryRows.DB.(*persistence.PostgresDB).Backend = rowsErrBackend
	storeRows := NewPersistenceModerationStore(factoryRows, nil, nil)
	if _, err := storeRows.SearchTags(ctx, TagSearchFilter{}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("search tags rows error = %v", err)
	}

	// 公告能力委托给 system 模块（非 nil 即走委托分支）。
	announcementStore := systemmodule.NewServiceWithFactory(nil, appconfig.NewStaticSource(config.Load()))
	delegating := NewPersistenceModerationStore(nil, announcementStore, nil)
	if _, err := delegating.SearchAnnouncements(ctx, systemmodule.AnnouncementSearchFilter{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("delegated search announcements = %v", err)
	}
}

// covMomentTagStore 注入标签关联查询错误。
type covMomentTagStore struct{}

func (covMomentTagStore) HasTagAssignments(context.Context, string) (bool, error) {
	return false, errCovAdmin
}

func (covMomentTagStore) DeleteTagAssignments(context.Context, string) error {
	return errCovAdmin
}

func TestNullableStringAndHelpers(t *testing.T) {
	if nullableString(nil) != nil {
		t.Fatal("nullableString(nil) should be nil")
	}
	if nullableString("") != nil || nullableString("  ") != nil {
		t.Fatal("blank strings should map to nil")
	}
	if got := nullableString("value"); got != "value" {
		t.Fatalf("nullableString = %v", got)
	}
	if got := nullableString(strPtr("value")); got != "value" {
		t.Fatalf("nullableString(ptr) = %v", got)
	}
	if nullableString(strPtr("")) != nil {
		t.Fatal("blank ptr should map to nil")
	}
	if strPtr("") != nil {
		t.Fatal("strPtr(empty) should be nil")
	}
	if got := strPtr("x"); got == nil || *got != "x" {
		t.Fatalf("strPtr = %v", got)
	}
	// 超管角色的权限快照固定为通配符。
	if perms := (adminListRecord{GrantAllPermissions: true}).permissionsForResponse(); len(perms) != 1 || perms[0] != wildcardPermission {
		t.Fatalf("permissionsForResponse = %#v", perms)
	}
}

var _ = systemmodule.AnnounceLevelInfo
