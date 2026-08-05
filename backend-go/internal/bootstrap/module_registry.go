package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/auth"
	"hotspot/backend-go/internal/modules/feed"
	"hotspot/backend-go/internal/modules/groupmessages"
	"hotspot/backend-go/internal/modules/groups"
	mapmodule "hotspot/backend-go/internal/modules/map"
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
	imageResolver := NewMomentImageResolver(mediaModule.Service)
	feedImageResolver := NewFeedImageResolver(mediaModule.Service)
	momentsModule := moments.NewModule(factory, usersModule.Service, sharedAppConfig, imageResolver)
	systemModule := system.NewModule(factory, sharedAppConfig)
	systemModule.SetHealthRedisPinger(redisPinger)

	notificationService := notify.NewServiceWithRepo(notify.NewPersistenceRepository(factory))
	shareModule := share.NewModule(factory)

	gmModule := groupmessages.NewModule(factory, usersModule.Service, notificationService, shareModule.Service)
	pcModule := privatechats.NewModule(factory, sharedAppConfig, shareModule.Service)

	systemModule.Service().SetBootstrapResolver(systemBootstrapResolver{
		users:  usersModule.Service,
		notify: notificationService,
	})

	reactionProviders := []reactions.TargetProvider{
		momentReactionProvider{moments: momentsModule.Service},
		commentReactionProvider{comments: momentsModule.CommentsService()},
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
	feedModule := feed.NewModule(factory, sharedAppConfig, feedImageResolver)
	feedModule.Service.SetReactionAggregator(feedReactionAggregator{reactions: reactionsModule.Service})
	// reference_point 回退链第 2 级:已登录用户最近位置(users 模块实现,经 Service 接口注入)
	feedModule.Service.SetLocationProvider(usersModule.Service)
	momentsModule.Service.SetLocationProvider(usersModule.Service)
	groupsModule.Service.SetLocationProvider(usersModule.Service)
	groupsModule.Service.SetAppConfig(sharedAppConfig)
	searchModule := search.NewModule(factory, sharedAppConfig)
	searchModule.Service.SetLocationProvider(usersModule.Service)
	// Construct auth + admin before the module slice so the admin repo's
	// IsBanned can be wired into the platform auth Service (BanChecker),
	// enforcing 10201 for banned accounts on RequiredAuth endpoints.
	authModule := auth.NewModule(ctx, cfg, factory, sharedAppConfig)
	adminModule := admin.NewModule(factory, sharedAppConfig)
	if svc := platformauth.DefaultService(); svc != nil {
		svc.SetBanChecker(adminModule.Repo)
	}
	return []modulekit.Module{
		adminModule,
		authModule,
		feedModule,
		gmModule,
		groupsModule,
		mapmodule.NewModule(factory),
		mediaModule,
		momentsModule,
		notify.NewModuleWithService(notificationService),
		pcModule,
		reactionsModule,
		reports.NewModule(factory),
		searchModule,
		shareModule,
		systemModule,
		usersModule,
	}
}
