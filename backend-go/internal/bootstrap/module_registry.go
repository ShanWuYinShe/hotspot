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
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
)

func BuildModules(cfg config.Config, factory *persistence.Factory) []modulekit.Module {
	return BuildModulesWithRedisPinger(cfg, factory, nil)
}

// BuildModulesWithRedisPinger is like BuildModules but additionally wires a
// redis liveness probe into the system module's health check. Pass nil when
// redis is not configured (single-instance mode) — the health endpoint then
// skips the redis probe instead of reporting degraded.
func BuildModulesWithRedisPinger(cfg config.Config, factory *persistence.Factory, redisPinger func(ctx context.Context) error) []modulekit.Module {
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

	gmModule := groupmessages.NewModule(factory, usersModule.Service)
	pcModule := privatechats.NewModule(factory, sharedAppConfig)

	notificationService := notify.NewServiceWithRepo(notify.NewPersistenceRepository(factory))
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
	// Wire the reactions aggregator into moments/comments so MomentItem.reactions[]
	// and CommentItem.reactions[] get filled at read time. Done after both
	// modules exist to break the import cycle.
	momentsModule.Service.SetReactionAggregator(momentReactionAggregator{reactions: reactionsModule.Service})
	momentsModule.CommentsService().SetReactionAggregator(commentReactionAggregator{reactions: reactionsModule.Service})
	feedModule := feed.NewModule(factory, sharedAppConfig, feedImageResolver)
	feedModule.Service.SetReactionAggregator(feedReactionAggregator{reactions: reactionsModule.Service})
	return []modulekit.Module{
		admin.NewModule(factory, sharedAppConfig),
		auth.NewModule(cfg, factory, sharedAppConfig),
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
		search.NewModule(factory, sharedAppConfig),
		share.NewModule(factory),
		systemModule,
		usersModule,
	}
}
