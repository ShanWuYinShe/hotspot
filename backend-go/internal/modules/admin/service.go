package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Service struct {
	store    Store
	provider Provider
}

func NewService(appCfg appconfig.Source) *Service {
	return NewServiceWithDeps(NewInMemoryStore(), NewAppConfigProvider(appCfg))
}

func NewServiceWithDeps(store Store, provider Provider) *Service {
	return &Service{store: store, provider: provider}
}

func (s *Service) GetMe(ctx context.Context) AdminPermissions {
	return s.store.GetMe(ctx)
}

func (s *Service) GetAppConfig(ctx context.Context) any {
	return s.provider.GetAppConfig(ctx)
}
