package appconfig

type GeoPoint struct {
	Latitude  float64
	Longitude float64
}

type Location struct {
	Point        GeoPoint
	LocationName string
}

type Features struct {
	EnablePublicRegistration bool
	EnableGuestLogin         bool
	EnableMoments            bool
	EnableNearbyPeople       bool
}

type Limits struct {
	MaxGroupMembers            int
	MaxMomentTags              int
	MaxMomentImages            int
	MomentDescriptionMaxLength int
	CommentMaxLength           int
	SearchRadiusMax            int
}

type Contact struct {
	SupportEmail      string
	PrivacyPolicyURL  string
	TermsOfServiceURL string
	OfficialWebsite   string
}

type Defaults struct {
	DefaultLocation    Location
	DefaultUserAvatar  string
	DefaultGroupAvatar string
}

type ResponseDefaults struct {
	SystemAnnouncementsCount int
	ChatRecentMessagesCount  int
}

type AppConfig struct {
	Features         Features
	Limits           Limits
	Contact          Contact
	Defaults         Defaults
	ResponseDefaults ResponseDefaults
}
