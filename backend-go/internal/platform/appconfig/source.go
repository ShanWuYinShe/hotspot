package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
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
	cfg      config.Config
	factory  *persistence.Factory
	fallback *memoryStore
	// loadMu 保护 loadAttempted/loadErr/loadedCfg：替代此前的 sync.Once——
	// Once 使启动瞬间的 DB 抖动把失败永久固化，进程余生只用默认配置且永不
	// 重试（限流阈值、功能开关、上传限制全部失真）。
	loadMu          sync.Mutex
	loadAttempted   bool
	loadErr         error
	lastLoadFailure time.Time
	loadedCfg       AppConfig
	// lastLoadSuccess 支撑成功后的周期性重载（reloadTTL）：此前成功加载一次后
	// 永不再查 DB，「配置变更即时生效」只在本进程 Update 时成立——多实例
	// 部署下其它实例永不感知 admin 写入 DB 的配置变更。
	lastLoadSuccess time.Time
}

// loadRetryBackoff 控制失败后的重试节奏：首次失败后按此间隔退避重试，避免
// 每次请求都打 DB。
const loadRetryBackoff = 30 * time.Second

// reloadTTL 是成功加载后的重查间隔：让「AppConfig 变更即时生效」的承诺
// （AGENTS「配置」节）在多实例部署下同样成立——其它实例最迟一个 TTL 内
// 感知写入 DB 的新配置。每 TTL 一次 QueryRow，开销可忽略。
const reloadTTL = 30 * time.Second

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
	s.loadMu.Lock()
	defer s.loadMu.Unlock()
	if !s.loadAttempted {
		s.loadAttempted = true
		s.tryLoad(ctx)
		return
	}
	// 加载失败按退避重试（此前 Once 语义下失败永不重试）。
	if s.loadErr != nil && time.Since(s.lastLoadFailure) >= loadRetryBackoff {
		s.tryLoad(ctx)
		return
	}
	// 成功后周期性重载：多实例部署下感知其它实例/admin 写入 DB 的配置变更
	//（此前成功一次后永不再查，「即时生效」仅单实例语义）。
	if s.loadErr == nil && time.Since(s.lastLoadSuccess) >= reloadTTL {
		s.tryLoad(ctx)
	}
}

// tryLoad 尝试从 DB 加载配置；调用方必须持有 loadMu。
func (s *PersistenceSource) tryLoad(ctx context.Context) {
	if db, ok := s.queryable(); ok {
		cfg, ok, err := loadFromDB(ctx, db)
		if err != nil {
			s.loadErr = err
			s.lastLoadFailure = time.Now()
			observability.L().Error("appconfig: load from db failed, using defaults; will retry", "error", err)
			return
		}
		s.loadErr = nil
		// 行不存在（新库）也算一次成功加载：记录 lastLoadSuccess，避免
		// 每次请求都因「无行」重查 DB。
		s.lastLoadSuccess = time.Now()
		if ok {
			s.loadedCfg = cfg
			s.fallback.Set(cfg)
			return
		}
	}
	s.fallback.Set(s.loadedCfg)
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
			EnableGuestRegister:      true,
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
			AuthSigninPer10Minutes:   60,
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
			WSHandshakePerMinute:     60,
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
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}
