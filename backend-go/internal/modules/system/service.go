package system

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Service struct {
	appConfig appconfig.Source
}

func NewService(appConfig appconfig.Source) *Service {
	return &Service{appConfig: appConfig}
}

func (s *Service) GetMeta() ApiVersionMeta {
	return ApiVersionMeta{
		Version: "v1",
		Status:  ApiVersionStatusActive,
	}
}

func (s *Service) Health() (int, string, *HealthStatus) {
	return 0, "", &HealthStatus{
		Status:    SystemHealthHealthy,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *Service) ListAnnouncements() SystemAnnouncementList {
	return SystemAnnouncementList{Items: []SystemAnnouncementItem{}}
}

func (s *Service) GetClientConfig() ClientConfig {
	cfg := s.appConfig.Get(context.Background())
	return ClientConfig{
		Features: FeatureSwitches{
			EnablePublicRegistration: cfg.Features.EnablePublicRegistration,
			EnableGuestLogin:         cfg.Features.EnableGuestLogin,
			EnableMoments:            cfg.Features.EnableMoments,
			EnableNearbyPeople:       cfg.Features.EnableNearbyPeople,
		},
		Limits: AppLimits{
			MaxGroupMembers:            cfg.Limits.MaxGroupMembers,
			MaxMomentTags:              cfg.Limits.MaxMomentTags,
			MaxMomentImages:            cfg.Limits.MaxMomentImages,
			MomentDescriptionMaxLength: cfg.Limits.MomentDescriptionMaxLength,
			CommentMaxLength:           cfg.Limits.CommentMaxLength,
			SearchRadiusMax:            cfg.Limits.SearchRadiusMax,
		},
		Contact: ContactInfo{
			SupportEmail:      cfg.Contact.SupportEmail,
			PrivacyPolicyURL:  cfg.Contact.PrivacyPolicyURL,
			TermsOfServiceURL: cfg.Contact.TermsOfServiceURL,
			OfficialWebsite:   cfg.Contact.OfficialWebsite,
		},
		Defaults: ClientDefaults{
			DefaultLocation: Location{
				Point: GeoPoint{
					Latitude:  cfg.Defaults.DefaultLocation.Point.Latitude,
					Longitude: cfg.Defaults.DefaultLocation.Point.Longitude,
				},
				LocationName: cfg.Defaults.DefaultLocation.LocationName,
			},
			DefaultUserAvatar:  cfg.Defaults.DefaultUserAvatar,
			DefaultGroupAvatar: cfg.Defaults.DefaultGroupAvatar,
		},
	}
}
