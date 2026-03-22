package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Provider interface {
	GetAppConfig(context.Context) appconfig.AppConfig
}

type AppConfigProvider struct {
	source appconfig.Source
}

func NewAppConfigProvider(source appconfig.Source) *AppConfigProvider {
	return &AppConfigProvider{source: source}
}

func (p *AppConfigProvider) GetAppConfig(ctx context.Context) appconfig.AppConfig {
	return p.source.Get(ctx)
}
