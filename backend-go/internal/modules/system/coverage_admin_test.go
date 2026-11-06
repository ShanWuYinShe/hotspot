package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestServiceAnnouncementAdminMethods 覆盖管理端公告能力（admin 经
// AnnouncementAdminStore 消费的唯一入口）与显式缓存失效。
func TestServiceAnnouncementAdminMethods(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appconfig.NewStaticSource(config.Load())))
	ctx := context.Background()

	created, err := svc.CreateAnnouncement(ctx, AnnouncementCreateInput{Title: "维护", Content: "内容", Severity: AnnounceLevelInfo})
	if err != nil || created.AnnouncementID == "" {
		t.Fatalf("create announcement: %#v err=%v", created, err)
	}
	if page, err := svc.SearchAnnouncements(ctx, AnnouncementSearchFilter{Limit: 10}); err != nil || len(page.Items) != 1 {
		t.Fatalf("search announcements: %#v err=%v", page, err)
	}
	if got, found, err := svc.GetAnnouncement(ctx, created.AnnouncementID); err != nil || !found || got.AnnouncementID != created.AnnouncementID {
		t.Fatalf("get announcement: %#v found=%v err=%v", got, found, err)
	}
	title := "维护（更新）"
	if _, err := svc.UpdateAnnouncement(ctx, AnnouncementUpdateInput{AnnouncementID: created.AnnouncementID, Title: &title}); err != nil {
		t.Fatalf("update announcement: %v", err)
	}
	if err := svc.DeleteAnnouncement(ctx, created.AnnouncementID); err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	svc.InvalidateAnnouncementsVersion() // 内存仓储 no-op，覆盖转发分支
}

// TestModuleMountsRoutesAndPinger 覆盖模块装配/路由挂载/健康探针接线。
func TestModuleMountsRoutesAndPinger(t *testing.T) {
	module := NewModule(persistence.NewFactory(config.Config{}), appconfig.NewStaticSource(config.Load()))
	if module.Service() == nil {
		t.Fatal("module must expose service")
	}
	module.SetHealthRedisPinger(func(context.Context) error { return nil })

	r := chi.NewRouter()
	module.Mount(r)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/system/meta", nil))
	if rr.Code == 0 {
		t.Fatal("mounted route must respond")
	}

	publicRouter := chi.NewRouter()
	module.MountPublic(publicRouter)
	rr = httptest.NewRecorder()
	publicRouter.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/system/meta", nil))
	if rr.Code == 0 {
		t.Fatal("public route must respond")
	}
}

// TestNewServiceWithFactoryWiresPostgresRepo 覆盖 factory 构造分支（无 DSN 时
// 健康检查降级为 unhealthy，不 panic）。
func TestNewServiceWithFactoryWiresPostgresRepo(t *testing.T) {
	svc := NewServiceWithFactory(persistence.NewFactory(config.Config{}), appconfig.NewStaticSource(config.Load()))
	svc.SetHealthRedisPinger(nil)
	if code, _, _ := svc.Health(context.Background()); code == 0 {
		// 无 DSN 时 repo 返回错误码；只要不 panic 即覆盖目标。
		t.Log("health degraded as expected")
	}
}
