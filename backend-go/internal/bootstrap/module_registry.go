package bootstrap

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

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
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/types"
)

func BuildModules(cfg config.Config, factory *persistence.Factory) []modulekit.Module {
	modules, _ := BuildModulesWithRedisPinger(context.Background(), cfg, factory, nil)
	return modules
}

// BuildModulesWithRedisPinger is like BuildModules but additionally wires the
// redis client into the shared-state stores: the system module's health check
// probe, the auth module's Redis login-failure tracker (P2-2：多实例共享爆破
// 计数且重启不丢), and (via the caller handing this same client to
// idempotency/ratelimit) cross-instance idempotency and rate limiting. Pass
// nil when redis is not configured (single-instance mode) — those stores then
// keep their in-process implementations. ctx governs the lifetime of
// module-owned background goroutines (e.g. the auth login-tracker sweeper)
// and should be the application root context.
//
// The second return value is the admin repository exposed as the platform-wide
// ban checker (IsBanned); bootstrap hands it to the realtime handler so the WS
// handshake enforces bans exactly like the HTTP RequiredAuth middleware.
func BuildModulesWithRedisPinger(ctx context.Context, cfg config.Config, factory *persistence.Factory, rdb *redis.Client) ([]modulekit.Module, admin.Repository) {
	sharedAppConfig := appconfig.NewPersistenceSource(cfg, factory)
	ratelimit.SetSource(sharedAppConfig)
	// 通知服务全应用单一实例：此前 moments/groups/users/admin 各自再建一份
	// notify.Service（共 5 个实例并存），装配属 bootstrap 职责，现统一注入。
	notificationService := notify.NewServiceWithRepo(notify.NewPostgresRepository(factory))
	// 通知后台清理（NOTIFY_PRUNE_ENABLED，默认开启）：每 24h 删除已读超 90 天
	// 的通知，防止表无限增长；goroutine 绑定 ctx 随优雅停机退出。
	if cfg.NotifyPruneEnabled {
		notificationService.StartPruneSweeper(ctx)
	}
	usersModule := users.NewModule(factory, sharedAppConfig, notificationService)
	groupsModule := groups.NewModule(factory, usersModule.Service, notificationService)
	mediaModule := media.NewModule(cfg, factory)
	mediaModule.Service.SetAppConfig(sharedAppConfig)
	// 群头像归属校验 + 稳定代理路径转换（groups create/update；对齐 users 的
	// AccountAvatarMediaResolver 注入，groupsModule 创建在先，此处回填注入）。
	groupsModule.Service.SetGroupAvatarMediaResolver(mediaModule.Service)
	// 数据保留期清理（RETENTION_SWEEP_ENABLED，默认开启）：未完成上传会话
	// 超期后删对象+删行（孤儿文件此前永不清理）。
	if cfg.RetentionSweepEnabled {
		mediaModule.Service.StartRetentionSweeper(ctx)
	}
	imageResolver := NewMomentImageResolver(mediaModule.Service)
	momentsModule := moments.NewModule(factory, usersModule.Service, sharedAppConfig, imageResolver, notificationService)
	systemModule := system.NewModule(factory, sharedAppConfig)
	systemModule.SetHealthRedisPinger(redisPinger(rdb))

	shareModule := share.NewModule(factory)

	gmModule := groupmessages.NewModule(factory, usersModule.Service, notificationService, shareModule.Service)
	// 消息撤回时间窗接线（spec chat.md：AppConfig.limits.message_recall_window_seconds）
	gmModule.Service.SetAppConfig(sharedAppConfig)
	pcModule := privatechats.NewModule(factory, sharedAppConfig, shareModule.Service, usersModule.Service)

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
	// 消息 reactions 聚合下发（spec api_spec_chat.md：消息项 reactions 恒在
	// 数组，空数组 = 无）。reactionProviders 承担「对谁可反应」的正向鉴权，
	// 这里是反向读路径：聊天消息拉取批量取聚合（reacted_by_me 按查看者）。
	gmModule.Service.SetReactionsSource(groupMessageReactionsSource{svc: reactionsModule.Service})
	pcModule.SetMessageReactionsSource(privateMessageReactionsSource{svc: reactionsModule.Service})
	// reaction emoji 白名单校验依赖 AppConfig.limits.reaction_emojis
	reactionsModule.Service.SetAppConfig(sharedAppConfig)
	// Wire the reactions aggregator into moments/comments so MomentItem.reactions[]
	// and CommentItem.reactions[] get filled at read time. Done after both
	// modules exist to break the import cycle.
	momentsModule.Service.SetReactionAggregator(momentReactionAggregator{reactions: reactionsModule.Service})
	momentsModule.CommentsService().SetReactionAggregator(commentReactionAggregator{reactions: reactionsModule.Service})
	// 评论拉黑拦截（spec api_spec_authz.md comments/create：黑名单 → 10200 + blocked_by_user）。
	// fn 签名无 ctx，此处用独立超时上下文单查双向拉黑表；查询出错向上返回错误
	// （fail-closed，由 comments service 映射统一 10600），authorID 拉黑了 callerID
	// 时返回 blocked=true。
	momentsModule.CommentsService().SetBlockChecker(func(callerID, authorID string) (bool, error) {
		checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		authorBlockedCaller, _, err := usersModule.Service.BlockStateBetween(checkCtx, authorID, callerID)
		return authorBlockedCaller, err
	})
	// 动态收藏/reaction 拉黑拦截（对齐评论与私聊口径：双向拉黑即不可交互，
	// 防止 reaction 击穿拉黑关系触发对方通知）。fail-closed 同上。
	momentsModule.Service.SetBlockChecker(func(callerID, authorID string) (bool, error) {
		checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		callerBlocked, authorBlocked, err := usersModule.Service.BlockStateBetween(checkCtx, callerID, authorID)
		return callerBlocked || authorBlocked, err
	})
	// reference_point 回退链第 2 级：已登录用户最近位置（users 模块实现，
	// 经 Service 接口注入 moments fetch_nearby/fetch_following 与各 search 端点）。
	momentsModule.Service.SetLocationProvider(usersModule.Service)
	groupsModule.Service.SetLocationProvider(usersModule.Service)
	// users 个人主页聚合的跨模块读（fetch_groups/fetch_moments/list_reacted/
	// 共同群组数）：注入 groups/moments Service，替代 users repo 直查对方表。
	usersModule.Service.SetContentProviders(groupsModule.Service, momentsModule.Service)
	groupsModule.Service.SetAppConfig(sharedAppConfig)
	// groups.Update 的 读-改-再读 序列包事务（对齐 moments.Update 纪律）。
	groupsModule.Service.SetTxRunner(persistence.NewTxRunner(factory))
	searchModule := search.NewModule(factory, sharedAppConfig)
	searchModule.Service.SetLocationProvider(usersModule.Service)
	// Construct auth + admin before the module slice so the admin repo's
	// IsBanned can be wired into the platform auth Service (BanChecker),
	// enforcing httpx.CodeAccountBanned for banned accounts on RequiredAuth endpoints.
	authModule := auth.NewModule(ctx, cfg, factory, sharedAppConfig)
	// 数据保留期清理：已撤销会话（含 IP PII）与过期验证码行到期物理删除。
	if cfg.RetentionSweepEnabled {
		authModule.StartRetentionSweeper(ctx)
	}
	if rdb != nil {
		// 登录失败锁定切 Redis（P2-2）：多实例共享计数，重启不丢锁定状态。
		authModule.SetLoginTracker(auth.NewRedisLoginTracker(rdb))
	}
	// upgrade 凭证写入接线（spec api_spec_user.md → upgrade 契约级语义）：
	// users.Upgrade 在同一事务内写 auth_accounts 凭证与 user_type。
	usersModule.Service.SetCredentialWriter(upgradeCredentialWriter{svc: authModule.Service()}, persistence.NewTxRunner(factory))
	// update_account 的 avatar_file_id 归属校验接线（spec 10300）。
	usersModule.Service.SetAvatarMediaResolver(mediaModule.Service)
	adminModule := admin.NewModule(factory, sharedAppConfig, notificationService)
	// 举报快照源接线（spec api_spec_types.md ReportSnapshot：服务端生成）：
	// 群聊 + 私聊两个消息源在各自模块构造完成后注入 reports；mediaFiles 经
	// factory 解析媒体元数据，使 snapshot_media 为完整 MessageMedia。
	reportsModule := reports.NewModule(factory)
	reportsModule.Service.SetMessageSnapshotSource(reports.NewMessageSnapshotSource(gmModule.Service, pcModule.MessageService(), reportsMediaSource{svc: mediaModule.Service}))
	if svc := platformauth.DefaultService(); svc != nil {
		svc.SetBanChecker(adminModule.Repo)
		// 封禁缓存失效接线：admin 封禁/解封状态变更后即时失效平台 auth 的
		// 进程内缓存（banCacheTTL 窗口内的撤销延迟归零）。
		adminModule.Service.SetBanCacheInvalidator(svc.InvalidateBanCache)
	}
	platformauth.SetAdminChecker(adminModule.Repo)
	// 首管理员引导（ADMIN_BOOTSTRAP_USER_ID，见 config.AdminBootstrapUserID）：
	// admin_users 表为空时把指定已注册用户提权为超管，解决「授权端点要求已是
	// 管理员」的鸡生蛋死锁。仅此一次成立——已有任意（未撤销）管理员即为 no-op。
	// 查询/写入失败只记日志不阻断启动（新环境首次启动迁移刚跑完，表必为空态可用）。
	if cfg.AdminBootstrapUserID != "" {
		adminPage, err := adminModule.Repo.ListAdmins(ctx, types.PaginationParams{Limit: 1})
		switch {
		case err != nil:
			observability.L().Warn("admin bootstrap: 无法读取管理员列表，跳过首管理员引导", "error", err)
		case len(adminPage.Items) > 0:
			observability.L().Info("admin bootstrap: 已存在平台管理员，跳过引导")
		default:
			if grantErr := adminModule.Repo.GrantAdmin(ctx, admin.AdminGrantUpsert{UserID: cfg.AdminBootstrapUserID, GrantAllPermissions: true}); grantErr != nil {
				observability.L().Error("admin bootstrap: 首管理员提权失败（确认该用户已完成注册）",
					"user_id", cfg.AdminBootstrapUserID, "error", grantErr)
			} else {
				observability.L().Info("admin bootstrap: 已授予首个平台超管", "user_id", cfg.AdminBootstrapUserID)
			}
		}
	}
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
	}, adminModule.Repo
}
