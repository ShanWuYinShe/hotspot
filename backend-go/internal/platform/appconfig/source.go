package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type Source interface {
	Get(context.Context) AppConfig
}

type MutableSource interface {
	Source
	Update(context.Context, func(AppConfig) AppConfig) (AppConfig, error)
}

type queryable interface {
	QueryRow(context.Context, string, ...any) persistence.Row
	Exec(context.Context, string, ...any) error
}

type StaticSource struct {
	mu  sync.RWMutex
	cfg AppConfig
}

type PersistenceSource struct {
	cfg       config.Config
	factory   *persistence.Factory
	fallback  *memoryStore
	loadOnce  sync.Once
	loadErr   error
	loadedCfg AppConfig
}

type memoryStore struct {
	mu  sync.RWMutex
	cfg AppConfig
}

const (
	appConfigConfigID = "default"
	queryAppConfigGet = `
SELECT payload_json, updated_at, COALESCE(updated_by_user_id, '')
FROM app_configs
WHERE config_id = $1`
	queryAppConfigUpsert = `
INSERT INTO app_configs (config_id, payload_json, updated_at, updated_by_user_id)
VALUES ($1,$2,$3,$4)
ON CONFLICT (config_id) DO UPDATE
SET payload_json = EXCLUDED.payload_json,
    updated_at = EXCLUDED.updated_at,
    updated_by_user_id = EXCLUDED.updated_by_user_id`
)

func NewStaticSource(runtime config.Config) *StaticSource {
	return &StaticSource{cfg: defaultConfig(runtime)}
}

func NewPersistenceSource(runtime config.Config, factory *persistence.Factory) *PersistenceSource {
	return &PersistenceSource{
		cfg:       runtime,
		factory:   factory,
		fallback:  newMemoryStore(defaultConfig(runtime)),
		loadedCfg: defaultConfig(runtime),
	}
}

func (s *StaticSource) Get(_ context.Context) AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.cfg)
}

func (s *StaticSource) Update(_ context.Context, apply func(AppConfig) AppConfig) (AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cloneConfig(apply(s.cfg))
	return cloneConfig(s.cfg), nil
}

func (s *PersistenceSource) Get(ctx context.Context) AppConfig {
	s.ensureLoaded(ctx)
	return s.fallback.Get()
}

func (s *PersistenceSource) Update(ctx context.Context, apply func(AppConfig) AppConfig) (AppConfig, error) {
	s.ensureLoaded(ctx)
	current := s.fallback.Get()
	next := cloneConfig(apply(current))
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	next.UpdatedAt = updatedAt
	if db, ok := s.queryable(); ok {
		payload, err := json.Marshal(next)
		if err != nil {
			return AppConfig{}, err
		}
		if err := db.Exec(ctx, queryAppConfigUpsert, appConfigConfigID, string(payload), parseTimeOrNow(updatedAt), nullableString(next.UpdatedByUserID)); err != nil {
			return AppConfig{}, err
		}
	}
	s.fallback.Set(next)
	return cloneConfig(next), nil
}

func (s *PersistenceSource) ensureLoaded(ctx context.Context) {
	s.loadOnce.Do(func() {
		if db, ok := s.queryable(); ok {
			cfg, ok, err := loadFromDB(ctx, db)
			if err != nil {
				s.loadErr = err
				return
			}
			if ok {
				s.loadedCfg = cfg
				s.fallback.Set(cfg)
				return
			}
		}
		s.fallback.Set(s.loadedCfg)
	})
	if s.loadErr != nil {
		s.fallback.Set(s.loadedCfg)
	}
}

func (s *PersistenceSource) queryable() (queryable, bool) {
	if s.factory == nil || s.factory.DB == nil || s.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	db, ok := s.factory.DB.(queryable)
	return db, ok
}

func loadFromDB(ctx context.Context, db queryable) (AppConfig, bool, error) {
	var payload string
	var updatedAt time.Time
	var updatedBy string
	if err := db.QueryRow(ctx, queryAppConfigGet, appConfigConfigID).Scan(&payload, &updatedAt, &updatedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AppConfig{}, false, nil
		}
		return AppConfig{}, false, err
	}
	var cfg AppConfig
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return AppConfig{}, false, err
	}
	if cfg.UpdatedAt == "" {
		cfg.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	}
	if cfg.UpdatedByUserID == "" {
		cfg.UpdatedByUserID = updatedBy
	}
	return cfg, true, nil
}

func newMemoryStore(cfg AppConfig) *memoryStore {
	return &memoryStore{cfg: cloneConfig(cfg)}
}

func (s *memoryStore) Get() AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.cfg)
}

func (s *memoryStore) Set(cfg AppConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cloneConfig(cfg)
}

func defaultConfig(runtime config.Config) AppConfig {
	return AppConfig{
		Features: Features{
			EnablePublicRegistration: true,
			EnableGuestRegister:         true,
			EnableMoments:            true,
			EnableNearbyPeople:       true,
		},
		Limits: Limits{
			MaxGroupMembers:            500,
			MaxMomentTags:              10,
			MaxMomentImages:            9,
			MaxChatFiles:               9,
			MomentDescriptionMaxLength: 2000,
			CommentMaxLength:           500,
			SearchRadiusMax:            50000,
			MessageRecallWindowSeconds: 120,
			StrangerMessageDailyQuota:  10,
			ReactionEmojis:             []string{"👍", "❤️", "😂", "😮", "😢", "🙏"},
			GroupInviteTTLHours:        168,
			GroupApplicationTTLHours:   72,
		},
		Contact: Contact{
			SupportEmail:      runtime.SupportEmail,
			PrivacyPolicyURL:  runtime.PrivacyPolicyURL,
			TermsOfServiceURL: runtime.TermsOfServiceURL,
			OfficialWebsite:   runtime.OfficialWebsiteURL,
		},
		Upload: UploadConfig{
			ImageMaxSize:      10 * 1024 * 1024,
			VideoMaxSize:      100 * 1024 * 1024,
			AudioMaxSize:      20 * 1024 * 1024,
			FileMaxSize:       50 * 1024 * 1024,
			AllowedImageTypes: []string{"image/jpeg", "image/png", "image/webp"},
			AllowedVideoTypes: []string{"video/mp4", "video/quicktime"},
		},
		Defaults: Defaults{
			// 默认位置与 geo.DefaultFallbackLatitude/DefaultFallbackLongitude
			// (appConfig 缺失时的 reference_point 回退锚点)保持一致;
			// appconfig 不依赖 geo 包,故此处保留字面量。
			DefaultLocation: Location{
				Point: GeoPoint{
					Latitude:  35.154922,
					Longitude: 112.682524,
				},
				LocationName: "南天门",
			},
			DefaultUserAvatar:  runtime.DefaultUserAvatar,
			DefaultGroupAvatar: runtime.DefaultGroupAvatar,
			DefaultGroupSettings: GroupSettings{
				JoinMethod:        "direct",
				RequiresApproval:  false,
				IsSearchable:      true,
				MembersSearchable: true,
				MaxMembers:        500,
				Location: Location{
					Point: GeoPoint{
						Latitude:  35.154922,
						Longitude: 112.682524,
					},
					LocationName: "南天门",
				},
			},
		},
		RateLimits: RateLimitConfig{
			GlobalPerHour:            10000,
			UserPerHour:              1000,
			AuthRegisterPer10Minutes: 20,
			AuthSigninPer10Minutes:    60,
			AuthGuestPer10Minutes:    60,
			AuthRefreshPer10Minutes:  120,
			AuthResetPer10Minutes:    20,
			SearchPerMinute:          60,
			WritePerMinute:           30,
			MomentsCreatePerMinute:   10,
			CommentsCreatePerMinute:  20,
			ReportsCreatePerHour:     20,
			UploadPresignPerHour:     100,
			WSSubscribePerSecond:     10,
			AdminReadPerMinute:       120,
			AdminWritePerMinute:      60,
		},
		ResponseDefaults: ResponseDefaults{
			SearchSuggestCount:       5,
			HotMomentsCount:          20,
			ChatRecentMessagesCount:  20,
			GroupAnnouncementsCount:  10,
			SystemAnnouncementsCount: 10,
			CommentTopRepliesCount:   3,
			UserProfileMomentsCount:  6,
		},
	}
}

func cloneConfig(cfg AppConfig) AppConfig {
	cfg.Upload.AllowedImageTypes = append([]string(nil), cfg.Upload.AllowedImageTypes...)
	cfg.Upload.AllowedVideoTypes = append([]string(nil), cfg.Upload.AllowedVideoTypes...)
	cfg.Limits.ReactionEmojis = append([]string(nil), cfg.Limits.ReactionEmojis...)
	return cfg
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func parseTimeOrNow(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}
