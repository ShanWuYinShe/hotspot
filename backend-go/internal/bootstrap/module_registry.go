package bootstrap

import (
	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/auth"
	"hotspot/backend-go/internal/modules/feed"
	"hotspot/backend-go/internal/modules/groupmessages"
	"hotspot/backend-go/internal/modules/groups"
	mapmodule "hotspot/backend-go/internal/modules/map"
	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/notifications"
	"hotspot/backend-go/internal/modules/privatechats"
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
	sharedAppConfig := appconfig.NewPersistenceSource(cfg, factory)
	ratelimit.SetSource(sharedAppConfig)
	usersModule := users.NewModule(factory, sharedAppConfig)
	groupsModule := groups.NewModule(factory, usersModule.Service)
	mediaModule := media.NewModule(cfg, factory)
	imageResolver := NewMomentImageResolver(mediaModule.Service)
	feedImageResolver := NewFeedImageResolver(mediaModule.Service)
	momentsModule := moments.NewModule(factory, usersModule.Service, sharedAppConfig, imageResolver)
	userSearcher := users.NewSearchAdapter(usersModule.Service)
	groupSearcher := groups.NewSearchAdapter(groupsModule.Service)
	momentSearcher := moments.NewSearchAdapter(momentsModule.Service)
	return []modulekit.Module{
		admin.NewModule(factory, sharedAppConfig),
		auth.NewModule(cfg, factory, sharedAppConfig),
		feed.NewModule(factory, sharedAppConfig, feedImageResolver),
		groupmessages.NewModule(factory, usersModule.Service),
		groupsModule,
		mapmodule.NewModule(factory),
		mediaModule,
		momentsModule,
		notifications.NewModule(factory),
		privatechats.NewModule(factory, sharedAppConfig),
		reports.NewModule(factory),
		search.NewModuleWithSearchers(factory, sharedAppConfig, userSearcher, groupSearcher, momentSearcher),
		share.NewModule(factory),
		system.NewModule(factory, sharedAppConfig),
		usersModule,
	}
}
