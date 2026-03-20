package appconfig

import (
	"context"

	"hotspot/backend-go/internal/platform/config"
)

type Source interface {
	Get(context.Context) AppConfig
}

type StaticSource struct {
	cfg AppConfig
}

func NewStaticSource(runtime config.Config) *StaticSource {
	return &StaticSource{
		cfg: AppConfig{
			Features: Features{
				EnablePublicRegistration: true,
				EnableGuestLogin:         true,
				EnableMoments:            true,
				EnableNearbyPeople:       true,
			},
			Limits: Limits{
				MaxGroupMembers:            500,
				MaxMomentTags:              10,
				MaxMomentImages:            9,
				MomentDescriptionMaxLength: 2000,
				CommentMaxLength:           500,
				SearchRadiusMax:            50000,
			},
			Contact: Contact{
				SupportEmail:      runtime.SupportEmail,
				PrivacyPolicyURL:  runtime.PrivacyPolicyURL,
				TermsOfServiceURL: runtime.TermsOfServiceURL,
				OfficialWebsite:   runtime.OfficialWebsiteURL,
			},
			Defaults: Defaults{
				DefaultLocation: Location{
					Point: GeoPoint{
						Latitude:  35.154922,
						Longitude: 112.682524,
					},
					LocationName: "南天门",
				},
				DefaultUserAvatar:  runtime.DefaultUserAvatar,
				DefaultGroupAvatar: runtime.DefaultGroupAvatar,
			},
			ResponseDefaults: ResponseDefaults{
				SystemAnnouncementsCount: 20,
				ChatRecentMessagesCount:  20,
			},
		},
	}
}

func (s *StaticSource) Get(_ context.Context) AppConfig {
	return s.cfg
}
