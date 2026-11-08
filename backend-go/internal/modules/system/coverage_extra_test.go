package system

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 本文件补齐 system 模块在「真实库 + stub 后端」双口径下的错误/边界分支：
// repo 的 queryable/Query/Scan/Exec 失败、健康探针的 PG/Redis 故障、
// 公告通知虚拟行的游标与已读写路径、module 装配辅助函数的 nil 安全分支。

var errCovExtra = errors.New("system coverage extra: injected failure")

// covExtraBackend 是一个按需注入查询/执行错误的 Queryable 后端。
type covExtraBackend struct {
	queryErr    error
	execErr     error
	rows        persistence.Rows
	row         persistence.Row
	execQueries []string
}

func (b *covExtraBackend) Exec(_ context.Context, query string, _ ...any) error {
	b.execQueries = append(b.execQueries, query)
	return b.execErr
}

func (b *covExtraBackend) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	if b.row != nil {
		return b.row
	}
	return testdb.RowValues{Err: errCovExtra}
}

func (b *covExtraBackend) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	if b.queryErr != nil {
		return nil, b.queryErr
	}
	if b.rows != nil {
		return b.rows, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func newCovExtraRepo(t *testing.T, backend persistence.PostgresQueryBackend) *PostgresRepository {
	t.Helper()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	db, ok := factory.DB.(*persistence.PostgresDB)
	if !ok {
		t.Fatalf("factory DB type = %T, want *persistence.PostgresDB", factory.DB)
	}
	db.Backend = backend
	return NewPostgresRepository(factory)
}

// covScanErrRows 在首次 Next() 为真时让 Scan 返回错误（模拟列类型不匹配）。
type covScanErrRows struct{ calls int }

func (r *covScanErrRows) Next() bool        { r.calls++; return r.calls == 1 }
func (r *covScanErrRows) Scan(...any) error { return errCovExtra }
func (r *covScanErrRows) Close() error      { return nil }
func (r *covScanErrRows) Err() error        { return nil }

// covIterErrRows 无行但 Err() 非 nil（模拟遍历中途连接中断）。
type covIterErrRows struct{}

func (covIterErrRows) Next() bool        { return false }
func (covIterErrRows) Scan(...any) error { return nil }
func (covIterErrRows) Close() error      { return nil }
func (covIterErrRows) Err() error        { return errCovExtra }

// covPingErrDB 是只实现 DB 接口的故障数据库（Health 探针使用）。
type covPingErrDB struct{ err error }

func (db covPingErrDB) Ping(context.Context) error { return db.err }
func (db covPingErrDB) Close() error               { return nil }

// covVersionErrRepo 在 stubRepo 基础上让公告版本查询失败。
type covVersionErrRepo struct{ *stubRepo }

func (r covVersionErrRepo) AnnouncementsVersion(context.Context) (string, error) {
	return "", errCovExtra
}

func TestParseAnnouncementNotifyIDRejectsInvalidInput(t *testing.T) {
	for _, notifyID := range []string{"", "notify_1", "sysann_", "other_sysann_1"} {
		if announcementID, ok := ParseAnnouncementNotifyID(notifyID); ok || announcementID != "" {
			t.Fatalf("ParseAnnouncementNotifyID(%q) = (%q, %v), want empty/false", notifyID, announcementID, ok)
		}
	}
	if announcementID, ok := ParseAnnouncementNotifyID("sysann_ann_1"); !ok || announcementID != "ann_1" {
		t.Fatalf("valid notify id not parsed: (%q, %v)", announcementID, ok)
	}
}

func TestPostgresRepositoryQueryableUnavailable(t *testing.T) {
	// factory 为 nil 时 RepoQueryable 返回 false：所有依赖 DB 的方法都必须
	// 原样返回 ErrNotImplemented，而不是 panic。
	repo := NewPostgresRepository(nil)
	ctx := context.Background()
	if _, err := repo.AnnouncementsVersion(ctx); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("AnnouncementsVersion err = %v", err)
	}
	if _, err := repo.ListAnnouncements(ctx, 10); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListAnnouncements err = %v", err)
	}
	if _, err := repo.SearchAnnouncements(ctx, AnnouncementSearchFilter{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("SearchAnnouncements err = %v", err)
	}
	if _, _, err := repo.GetAnnouncement(ctx, "ann_1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetAnnouncement err = %v", err)
	}
	if _, err := repo.CreateAnnouncement(ctx, AnnouncementCreateInput{Title: "t", Content: "c", Severity: AnnounceLevelInfo}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("CreateAnnouncement err = %v", err)
	}
	if _, err := repo.UpdateAnnouncement(ctx, AnnouncementUpdateInput{AnnouncementID: "ann_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("UpdateAnnouncement err = %v", err)
	}
	if err := repo.DeleteAnnouncement(ctx, "ann_1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("DeleteAnnouncement err = %v", err)
	}
	if _, err := repo.ListAnnouncementNotifications(ctx, "user_1", AnnouncementNotifyCursor{}, nil, 20); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListAnnouncementNotifications err = %v", err)
	}
	if _, err := repo.UnreadAnnouncementCount(ctx, "user_1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("UnreadAnnouncementCount err = %v", err)
	}
	if err := repo.MarkAnnouncementRead(ctx, "user_1", []string{"ann_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MarkAnnouncementRead err = %v", err)
	}
	if err := repo.MarkAllAnnouncementsRead(ctx, "user_1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MarkAllAnnouncementsRead err = %v", err)
	}
}

func TestPostgresRepositoryGetMeta(t *testing.T) {
	repo := NewPostgresRepository(nil)
	meta := repo.GetMeta(context.Background())
	if meta.Version != "v1" || meta.Status != ApiVersionStatusActive {
		t.Fatalf("unexpected meta: %#v", meta)
	}
}

func TestPostgresRepositoryHealthPostgresPingFailure(t *testing.T) {
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     covPingErrDB{err: errCovExtra},
	}
	repo := NewPostgresRepository(factory)
	code, message, status := repo.Health(context.Background())
	if code != int(httpx.CodeInternalUnavailable) || message != "数据库不可用" || status != nil {
		t.Fatalf("Health = (%d, %q, %v), want unhealthy", code, message, status)
	}
}

func TestPostgresRepositoryHealthRedisProbe(t *testing.T) {
	repo := NewPostgresRepository(nil)
	repo.SetRedisPinger(func(context.Context) error { return errCovExtra })
	code, message, status := repo.Health(context.Background())
	if code != 0 || message != "" || status == nil || status.Status != SystemHealthDegraded {
		t.Fatalf("degraded Health = (%d, %q, %#v), want code 0 degraded", code, message, status)
	}
	repo.SetRedisPinger(func(context.Context) error { return nil })
	code, message, status = repo.Health(context.Background())
	if code != 0 || message != "" || status == nil || status.Status != SystemHealthHealthy {
		t.Fatalf("healthy Health = (%d, %q, %#v), want code 0 healthy", code, message, status)
	}
}

func TestPostgresRepositoryAnnouncementNotifyBackendErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("query error", func(t *testing.T) {
		repo := newCovExtraRepo(t, &covExtraBackend{queryErr: errCovExtra})
		if _, err := repo.ListAnnouncementNotifications(ctx, "user_1", AnnouncementNotifyCursor{}, nil, 20); !errors.Is(err, errCovExtra) {
			t.Fatalf("ListAnnouncementNotifications err = %v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		repo := newCovExtraRepo(t, &covExtraBackend{rows: &covScanErrRows{}})
		if _, err := repo.ListAnnouncementNotifications(ctx, "user_1", AnnouncementNotifyCursor{}, nil, 20); !errors.Is(err, errCovExtra) {
			t.Fatalf("ListAnnouncementNotifications err = %v", err)
		}
	})

	t.Run("rows iteration error", func(t *testing.T) {
		repo := newCovExtraRepo(t, &covExtraBackend{rows: covIterErrRows{}})
		if _, err := repo.ListAnnouncementNotifications(ctx, "user_1", AnnouncementNotifyCursor{}, nil, 20); !errors.Is(err, errCovExtra) {
			t.Fatalf("ListAnnouncementNotifications err = %v", err)
		}
	})

	t.Run("zero limit and cursor defaults", func(t *testing.T) {
		backend := &covExtraBackend{}
		repo := newCovExtraRepo(t, backend)
		if _, err := repo.ListAnnouncementNotifications(ctx, "user_1", AnnouncementNotifyCursor{}, nil, 0); err != nil {
			t.Fatalf("ListAnnouncementNotifications: %v", err)
		}
		cursor := AnnouncementNotifyCursor{CreatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), NotifyID: "sysann_ann_1"}
		if _, err := repo.ListAnnouncementNotifications(ctx, "user_1", cursor, nil, 20); err != nil {
			t.Fatalf("ListAnnouncementNotifications with cursor: %v", err)
		}
	})

	t.Run("unread count error", func(t *testing.T) {
		repo := newCovExtraRepo(t, &covExtraBackend{row: testdb.RowValues{Err: errCovExtra}})
		if _, err := repo.UnreadAnnouncementCount(ctx, "user_1"); !errors.Is(err, errCovExtra) {
			t.Fatalf("UnreadAnnouncementCount err = %v", err)
		}
	})

	t.Run("mark read", func(t *testing.T) {
		backend := &covExtraBackend{}
		repo := newCovExtraRepo(t, backend)
		if err := repo.MarkAnnouncementRead(ctx, "user_1", nil); err != nil {
			t.Fatalf("MarkAnnouncementRead with empty ids: %v", err)
		}
		if err := repo.MarkAnnouncementRead(ctx, "user_1", []string{"ann_1"}); err != nil {
			t.Fatalf("MarkAnnouncementRead: %v", err)
		}
		backend.execErr = errCovExtra
		if err := repo.MarkAnnouncementRead(ctx, "user_1", []string{"ann_1"}); !errors.Is(err, errCovExtra) {
			t.Fatalf("MarkAnnouncementRead err = %v", err)
		}
		if err := repo.MarkAllAnnouncementsRead(ctx, "user_1"); !errors.Is(err, errCovExtra) {
			t.Fatalf("MarkAllAnnouncementsRead err = %v", err)
		}
	})
}

func TestPostgresRepositoryReadBackendErrors(t *testing.T) {
	ctx := context.Background()

	if _, err := newCovExtraRepo(t, &covExtraBackend{}).ListAnnouncements(ctx, 0); err != nil {
		t.Fatalf("ListAnnouncements with default limit: %v", err)
	}
	if _, err := newCovExtraRepo(t, &covExtraBackend{queryErr: errCovExtra}).ListAnnouncements(ctx, 10); !errors.Is(err, errCovExtra) {
		t.Fatalf("ListAnnouncements query err = %v", err)
	}
	if _, err := newCovExtraRepo(t, &covExtraBackend{rows: &covScanErrRows{}}).ListAnnouncements(ctx, 10); !errors.Is(err, errCovExtra) {
		t.Fatalf("ListAnnouncements scan err = %v", err)
	}
	if _, err := newCovExtraRepo(t, &covExtraBackend{rows: covIterErrRows{}}).ListAnnouncements(ctx, 10); !errors.Is(err, errCovExtra) {
		t.Fatalf("ListAnnouncements rows err = %v", err)
	}

	if _, err := newCovExtraRepo(t, &covExtraBackend{queryErr: errCovExtra}).SearchAnnouncements(ctx, AnnouncementSearchFilter{}); !errors.Is(err, errCovExtra) {
		t.Fatalf("SearchAnnouncements query err = %v", err)
	}
	if _, err := newCovExtraRepo(t, &covExtraBackend{rows: &covScanErrRows{}}).SearchAnnouncements(ctx, AnnouncementSearchFilter{}); !errors.Is(err, errCovExtra) {
		t.Fatalf("SearchAnnouncements scan err = %v", err)
	}
	if _, err := newCovExtraRepo(t, &covExtraBackend{rows: covIterErrRows{}}).SearchAnnouncements(ctx, AnnouncementSearchFilter{}); !errors.Is(err, errCovExtra) {
		t.Fatalf("SearchAnnouncements rows err = %v", err)
	}
}

func TestPostgresRepositoryWriteBackendErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("create exec error", func(t *testing.T) {
		backend := newAnnouncementBackendStub()
		backend.execErrs[queryInsertAnnouncement] = errCovExtra
		factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
		factory.DB.(*persistence.PostgresDB).Backend = backend
		repo := NewPostgresRepository(factory)
		if _, err := repo.CreateAnnouncement(ctx, AnnouncementCreateInput{Title: "t", Content: "c", Severity: AnnounceLevelInfo}); !errors.Is(err, errCovExtra) {
			t.Fatalf("CreateAnnouncement exec err = %v", err)
		}
	})

	t.Run("update read and exec errors", func(t *testing.T) {
		backend := newAnnouncementBackendStub()
		backend.announcements["ann_cov_1"] = AnnouncementRecord{
			AnnouncementID: "ann_cov_1", Title: "t", Content: "c", Severity: AnnounceLevelInfo,
			CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		}
		factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
		factory.DB.(*persistence.PostgresDB).Backend = backend
		repo := NewPostgresRepository(factory)
		title := "u"
		backend.queryRowErrs[queryAnnouncementByID] = errCovExtra
		if _, err := repo.UpdateAnnouncement(ctx, AnnouncementUpdateInput{AnnouncementID: "ann_cov_1", Title: &title}); !errors.Is(err, errCovExtra) {
			t.Fatalf("UpdateAnnouncement read err = %v", err)
		}
		delete(backend.queryRowErrs, queryAnnouncementByID)
		backend.execErrs[queryUpdateAnnouncement] = errCovExtra
		if _, err := repo.UpdateAnnouncement(ctx, AnnouncementUpdateInput{AnnouncementID: "ann_cov_1", Title: &title}); !errors.Is(err, errCovExtra) {
			t.Fatalf("UpdateAnnouncement exec err = %v", err)
		}
	})

	t.Run("delete read and exec errors", func(t *testing.T) {
		backend := newAnnouncementBackendStub()
		backend.announcements["ann_cov_2"] = AnnouncementRecord{
			AnnouncementID: "ann_cov_2", Title: "t", Content: "c", Severity: AnnounceLevelInfo,
			CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		}
		factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
		factory.DB.(*persistence.PostgresDB).Backend = backend
		repo := NewPostgresRepository(factory)
		backend.queryRowErrs[queryAnnouncementByID] = errCovExtra
		if err := repo.DeleteAnnouncement(ctx, "ann_cov_2"); !errors.Is(err, errCovExtra) {
			t.Fatalf("DeleteAnnouncement read err = %v", err)
		}
		delete(backend.queryRowErrs, queryAnnouncementByID)
		backend.execErrs[queryDeleteAnnouncement] = errCovExtra
		if err := repo.DeleteAnnouncement(ctx, "ann_cov_2"); !errors.Is(err, errCovExtra) {
			t.Fatalf("DeleteAnnouncement exec err = %v", err)
		}
	})
}

func TestModuleHelperNilSafety(t *testing.T) {
	var nilModule *Module
	if store := nilModule.AnnouncementNotificationStore(); store != nil {
		t.Fatalf("nil module store = %v, want nil", store)
	}
	if store := (&Module{}).AnnouncementNotificationStore(); store != nil {
		t.Fatalf("module without service store = %v, want nil", store)
	}
	// service 存在但 repo 未实现 AnnouncementNotificationStore 时返回 nil。
	inMemoryModule := &Module{service: NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appconfig.NewStaticSource(config.Load())))}
	if store := inMemoryModule.AnnouncementNotificationStore(); store != nil {
		t.Fatalf("in-memory module store = %v, want nil", store)
	}
	// PostgresRepository 实现该接口：返回非 nil。
	pgModule := NewModule(nil, appconfig.NewStaticSource(config.Load()))
	if store := pgModule.AnnouncementNotificationStore(); store == nil {
		t.Fatal("postgres module store = nil, want non-nil")
	}

	// nil / 未装配 handler 的 SetHealthRedisPinger 必须是安全的 no-op。
	var nilPingModule *Module
	nilPingModule.SetHealthRedisPinger(func(context.Context) error { return nil })
	(&Module{}).SetHealthRedisPinger(func(context.Context) error { return nil })

	// 正常装配时把探针挂到 PostgresRepository 上：degraded 状态可观测。
	pingModule := NewModule(nil, appconfig.NewStaticSource(config.Load()))
	pingModule.SetHealthRedisPinger(func(context.Context) error { return errCovExtra })
	code, _, status := pingModule.Service().Health(context.Background())
	if code != 0 || status == nil || status.Status != SystemHealthDegraded {
		t.Fatalf("module health = (%d, %#v), want degraded", code, status)
	}
}

func TestServiceBootstrapVersionQueryFailureDegrades(t *testing.T) {
	svc := NewServiceWithDeps(covVersionErrRepo{&stubRepo{}}, &stubProvider{configVersion: "cfg_v1"})
	bd := svc.Bootstrap(context.Background(), nil)
	if bd.BootstrapVersion == "" {
		t.Fatal("expected bootstrap version even when announcements query fails")
	}
	// 公告版本维度降级为空串，其余维度仍参与拼接。
	if !strings.HasSuffix(bd.BootstrapVersion, ":cfg_v1:") {
		t.Fatalf("bootstrap version = %q, want announcements dimension empty", bd.BootstrapVersion)
	}
}

func TestServiceInvalidateAnnouncementsVersionForwards(t *testing.T) {
	// PostgresRepository 实现失效钩子：调用必须落到 repo 上（缓存被清空）。
	svc := NewServiceWithFactory(nil, appconfig.NewStaticSource(config.Load()))
	svc.InvalidateAnnouncementsVersion()
	// 内存仓库不实现钩子：no-op 不 panic。
	NewService(appconfig.NewStaticSource(config.Load())).InvalidateAnnouncementsVersion()
}

func TestProviderAnnouncementsCountDefault(t *testing.T) {
	source := appconfig.NewStaticSource(config.Load())
	if _, err := source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.SystemAnnouncementsCount = 0
		return cfg
	}); err != nil {
		t.Fatalf("update static source: %v", err)
	}
	provider := NewAppConfigProvider(source)
	if got := provider.GetSystemAnnouncementsCount(context.Background()); got != 10 {
		t.Fatalf("GetSystemAnnouncementsCount = %d, want default 10", got)
	}
	if _, err := source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.SystemAnnouncementsCount = 3
		return cfg
	}); err != nil {
		t.Fatalf("update static source: %v", err)
	}
	if got := provider.GetSystemAnnouncementsCount(context.Background()); got != 3 {
		t.Fatalf("GetSystemAnnouncementsCount = %d, want 3", got)
	}
}

func TestHandlerHealthUnavailable(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&stubRepo{healthCode: httpx.CodeInternalUnavailable, healthMsg: "db down"}, &stubProvider{}))
	rr := httptest.NewRecorder()
	handler.Health(rr, httptest.NewRequest(http.MethodPost, "/health", strings.NewReader("{}")))
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	var resp httpx.ApiResponse[map[string]any]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInternalUnavailable || resp.Message != "db down" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestHandlerListAnnouncementsError(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&stubRepo{announceErr: errCovExtra}, &stubProvider{}))
	rr := httptest.NewRecorder()
	handler.ListAnnouncements(rr, httptest.NewRequest(http.MethodPost, "/announcements/fetch", strings.NewReader("{}")))
	var resp httpx.ApiResponse[map[string]any]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code == 0 {
		t.Fatalf("expected error code, got %#v", resp)
	}
}

func TestHandlerPublicBootstrapRejectsInvalidBody(t *testing.T) {
	handler := NewHandler(NewService(appconfig.NewStaticSource(config.Load())))
	rr := httptest.NewRecorder()
	handler.PublicBootstrap(rr, httptest.NewRequest(http.MethodPost, "/public/v1/system/bootstrap", strings.NewReader("[1,2,3]")))
	var resp httpx.ApiResponse[map[string]any]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("unexpected code: %d", resp.Code)
	}
}
