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
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/persistence"
)

func BuildModules(cfg config.Config, factory *persistence.Factory) []modulekit.Module {
	return []modulekit.Module{
		admin.NewModule(cfg),
		auth.NewModule(cfg, factory),
		feed.NewModule(),
		groupmessages.NewModule(factory),
		groups.NewModule(),
		mapmodule.NewModule(),
		media.NewModule(),
		moments.NewModule(factory),
		notifications.NewModule(),
		privatechats.NewModule(cfg, factory),
		reports.NewModule(),
		search.NewModule(),
		share.NewModule(),
		system.NewModule(cfg),
		users.NewModule(factory),
	}
}
