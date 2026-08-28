package bootstrap

import (
	"context"
	"time"

	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/auth"
	"hotspot/backend-go/internal/modules/groupmessages"
	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/mapmodule"
	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/privatechats"
	"hotspot/backend-go/internal/modules/reactions"
	"hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/modules/search"
	"hotspot/backend-go/internal/modules/share"
	"hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
)

func BuildModules(cfg config.Config, factory *persistence.Factory) []modulekit.Module {
	return BuildModulesWithRedisPinger(context.Background(), cfg, factory, nil)
}

// BuildModulesWithRedisPinger is like BuildModules but additionally wires a
// redis liveness probe into the system module's health check. Pass nil when
// redis is not configured (single-instance mode) — the health endpoint then
// skips the redis probe instead of reporting degraded. ctx governs the
// lifetime of module-owned background goroutines (e.g. the auth login-tracker
// sweeper) and should be the application root context.
func BuildModulesWithRedisPinger(ctx context.Context, cfg config.Config, factory *persistence.Factory, redisPinger func(ctx context.Context) error) []modulekit.Module {
	sharedAppConfig := appconfig.NewPersistenceSource(cfg, factory)
	ratelimit.SetSource(sharedAppConfig)
	usersModule := users.NewModule(factory, sharedAppConfig)
	groupsModule := groups.NewModule(factory, usersModule.Service)
	mediaModule := media.NewModule(cfg, factory)
	mediaModule.Service.SetAppConfig(sharedAppConfig)
	imageResolver := NewMomentImageResolver(mediaModule.Service)
	momentsModule := moments.NewModule(factory, usersModule.Service, sharedAppConfig, imageResolver)
	systemModule := system.NewModule(factory, sharedAppConfig)
	systemModule.SetHealthRedisPinger(redisPinger)

	notificationService := notify.NewServiceWithRepo(notify.NewPostgresRepository(factory))
	shareModule := share.NewModule(factory)

	gmModule := groupmessages.NewModule(factory, usersModule.Service, notificationService, shareModule.Service)
	pcModule := privatechats.NewModule(factory, sharedAppConfig, shareModule.Service)

	systemModule.Service().SetBootstrapResolver(systemBootstrapResolver{
		users:  usersModule.Service,
		notify: notificationService,
	})

	reactionProviders := []reactions.TargetProvider{
		momentReactionProvider{moments: momentsModule.Service},
		commentReactionProvider{comments: momentsModule.CommentsService(), moments: momentsModule.Service},
		messageReactionProvider{groupMessages: gmModule.Service, privateMessages: pcModule.MessageService()},
	}
	reactionsModule := reactions.NewModule(factory, reactionProviders, notificationService, usersModule.Service)
	// reaction emoji 白名单校验依赖 AppConfig.limits.reaction_emojis
	reactionsModule.Service.SetAppConfig(sharedAppConfig)
	// Wire the reactions aggregator into moments/comments so MomentItem.reactions[]
	// and CommentItem.reactions[] get filled at read time. Done after both
	// modules exist to break the import cycle.
	momentsModule.Service.SetReactionAggregator(momentReactionAggregator{reactions: reactionsModule.Service})
	momentsModule.CommentsService().SetReactionAggregator(commentReactionAggregator{reactions: reactionsModule.Service})
	// 评论拉黑拦截（spec api_spec_authz.md comments/create：黑名单 → 10200 + blocked_by_user）。
	// fn 签名无 ctx，此处用独立超时上下文单查双向拉黑表；查询出错放行（fail-open，
	// 与 resourceguard 口径一致），authorID 拉黑了 callerID 时返回 true。
	momentsModule.CommentsService().SetBlockChecker(func(callerID, authorID string) bool {
		checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		authorBlockedCaller, _, err := usersModule.Service.BlockStateBetween(checkCtx, authorID, callerID)
		return err == nil && authorBlockedCaller
	})
	// reference_point 回退链第 2 级：已登录用户最近位置（users 模块实现，
	// 经 Service 接口注入 moments fetch_nearby/fetch_following 与各 search 端点）。
	momentsModule.Service.SetLocationProvider(usersModule.Service)
	groupsModule.Service.SetLocationProvider(usersModule.Service)
	groupsModule.Service.SetAppConfig(sharedAppConfig)
	searchModule := search.NewModule(factory, sharedAppConfig)
	searchModule.Service.SetLocationProvider(usersModule.Service)
	// Construct auth + admin before the module slice so the admin repo's
	// IsBanned can be wired into the platform auth Service (BanChecker),
	// enforcing httpx.CodeAccountBanned for banned accounts on RequiredAuth endpoints.
	authModule := auth.NewModule(ctx, cfg, factory, sharedAppConfig)
	adminModule := admin.NewModule(factory, sharedAppConfig)
	// 举报快照源接线（spec api_spec_types.md ReportSnapshot：服务端生成）：
	// 群聊 + 私聊两个消息源在各自模块构造完成后注入 reports；mediaFiles 经
	// factory 解析媒体元数据，使 snapshot_media 为完整 MessageMedia。
	reportsModule := reports.NewModule(factory)
	reportsModule.Service.SetMessageSnapshotSource(newReportMessageSnapshotSource(factory, gmModule.Service, pcModule.MessageService()))
	if svc := platformauth.DefaultService(); svc != nil {
		svc.SetBanChecker(adminModule.Repo)
	}
	platformauth.SetAdminChecker(adminModule.Repo)
	// Owner-dismiss audit entries are written through the admin module (spec
	// api_spec_group.md: dismiss reason is audit-only).
	groupsModule.Service.SetGroupDismissAuditWriter(adminModule)
	// 守卫子树依赖接线（spec api_spec_authz.md）：
	// - 被屏蔽标签检查（groups/create tags → 10400，admin 实现）
	// - 管理员撤回任意成员消息（groups/admin/messages/recall，groupmessages 实现）
	groups.SetBlockedTagChecker(adminModule.BlockedTagChecker())
	groups.RecallMessageAsAdmin = func(ctx context.Context, groupID, messageID, operatorID string) error {
		return gmModule.Service.RecallMessageAsAdmin(ctx, operatorID, groupID, messageID)
	}
	return []modulekit.Module{
		adminModule,
		authModule,
		gmModule,
		groupsModule,
		mapmodule.NewModule(factory),
		mediaModule,
		momentsModule,
		notify.NewModuleWithService(notificationService),
		pcModule,
		reactionsModule,
		reportsModule,
		searchModule,
		shareModule,
		systemModule,
		usersModule,
	}
}
