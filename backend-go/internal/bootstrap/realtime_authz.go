package bootstrap

import (
	"context"
	"encoding/json"
	"time"

	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/hotcache"
	"hotspot/backend-go/internal/platform/realtime"
)

// realtimeBanAuthorizer adapts the admin module repository to the realtime
// handler's ban check at WS auth time (parity with HTTP RequiredAuth).
type realtimeBanAuthorizer struct{ repo admin.Repository }

func (a realtimeBanAuthorizer) IsBanned(ctx context.Context, userID string) (bool, error) {
	return a.repo.IsBanned(ctx, userID)
}

// BanDetails 提供封禁详情（realtime 的 AccountBanned 事件负载，spec
// api_spec_websocket.md）：限期封禁换算 banned_until，永久封禁为 nil。
func (a realtimeBanAuthorizer) BanDetails(ctx context.Context, userID string) (realtime.BanDetails, bool, error) {
	state, found, err := a.repo.GetUserBanState(ctx, userID)
	if err != nil || !found {
		return realtime.BanDetails{}, false, err
	}
	detail := realtime.BanDetails{
		Reason:       state.Reason,
		DurationDays: state.DurationDays,
		BannedAt:     state.BannedAt,
	}
	// 与 UserBanState.Expired 同口径：DurationDays>0 且 BannedAt 非零为限期封禁。
	if state.DurationDays > 0 && !state.BannedAt.IsZero() {
		until := state.BannedAt.Add(time.Duration(state.DurationDays) * 24 * time.Hour)
		detail.BannedUntil = &until
	}
	return detail, true, nil
}

// realtimeActorProvider 只保留 typing actor 解析：订阅鉴权已收敛到
// userRepo.GetSettings（一次查询），不再经全量 OpenTheDoor 取 presence。
type realtimeActorProvider struct{ svc *users.Service }

// cachedUserSettings 是 GetSettings 缓存值的最小投影（JSON 存 hotcache）。
type cachedUserSettings struct {
	Presence bool   `json:"p"`
	Allow    string `json:"a"`
	Found    bool   `json:"f"`
}

type realtimeUserRepo struct {
	repo  users.Repository
	cache *hotcache.Cache
}

// NewRealtimeUserRepo 构造带设置热缓存的 realtime 用户鉴权适配器
// （typing 每帧一次 GetSettings；缓存键与 users 模块写失效共享同一格式）。
func NewRealtimeUserRepo(repo users.Repository, cache *hotcache.Cache) realtimeUserRepo {
	return realtimeUserRepo{repo: repo, cache: cache}
}

// realtimeGroupRepo 群成员鉴权适配器：typing 每帧都要 IsMember，经 hotcache
// 读穿缓存（TTL 60s + groups 写路径删键）挡掉大部分 DB 往返；cache 可为 nil
// （未配 Redis 的本地降级下 hotcache 仍可用进程内 map，New 不返回 nil）。
type realtimeGroupRepo struct {
	repo  groups.Repository
	cache *hotcache.Cache
}

// NewRealtimeGroupRepo 构造带成员热缓存的 realtime 群鉴权适配器。
func NewRealtimeGroupRepo(repo groups.Repository, cache *hotcache.Cache) realtimeGroupRepo {
	return realtimeGroupRepo{repo: repo, cache: cache}
}

// realtimePresenceWriter 把 users repo 的运行时 presence 写（WS 连接生命周期）
// 适配到 realtime 的 presenceWriter 接口。repo 用具体类型 *users.PostgresRepository：
// SetPresenceStatus 只挂在 Postgres 实现（memory repo 无 WS 生命周期，无需实现）。
type realtimePresenceWriter struct{ repo *users.PostgresRepository }

func (a realtimePresenceWriter) SetPresence(ctx context.Context, userID string, online bool) error {
	status := users.UserPresenceOffline
	if online {
		status = users.UserPresenceOnline
	}
	return a.repo.SetPresenceStatus(ctx, userID, status)
}

func (a realtimeActorProvider) GetActorInfo(ctx context.Context, userID string) (actor.Info, error) {
	return a.svc.GetActorInfo(ctx, userID)
}

func (a realtimeUserRepo) GetSettings(ctx context.Context, userID string) (bool, string, bool, error) {
	key := hotcache.UserSettingsKey(userID)
	if a.cache != nil {
		if v, ok := a.cache.Get(ctx, key); ok {
			var cs cachedUserSettings
			if json.Unmarshal([]byte(v), &cs) == nil {
				return cs.Presence, cs.Allow, cs.Found, nil
			}
		}
	}
	settings, ok, err := a.repo.GetSettings(ctx, userID)
	if err != nil {
		return false, "", false, err
	}
	if a.cache != nil {
		if cs, err := json.Marshal(cachedUserSettings{Presence: settings.PresenceVisible, Allow: string(settings.AllowPrivateChat), Found: ok}); err == nil {
			a.cache.Set(ctx, key, string(cs), hotcache.UserSettingsTTL)
		}
	}
	return settings.PresenceVisible, string(settings.AllowPrivateChat), ok, err
}

// GetSettingsBatch 批量 presence/私聊设置（realtime 订阅鉴权批量路径）。
// 返回轻量结构避免 realtime 依赖 users 的完整 settings 类型。
func (a realtimeUserRepo) GetSettingsBatch(ctx context.Context, userIDs []string) (map[string]realtime.PresenceSetting, error) {
	settings, err := a.repo.GetSettingsBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]realtime.PresenceSetting, len(settings))
	for id, s := range settings {
		out[id] = realtime.PresenceSetting{PresenceVisible: s.PresenceVisible, Found: true}
	}
	return out, nil
}

// cachedRelPair 是 Relationship 缓存值的最小投影（关注方向；拉黑在 BlockPairKey）。
type cachedRelPair struct {
	Following  bool `json:"f"`
	FollowedBy bool `json:"b"`
}

func (a realtimeUserRepo) Relationship(ctx context.Context, userID, targetUserID string) (bool, bool, bool, error) {
	// 缓存值按【归一化后】参数序存储（RelPairKey 归一），读侧按 swapped 还原
	// 实参方向——否则 a>b 时关注方向会翻转。
	key := hotcache.RelPairKey(userID, targetUserID)
	_, swapped := hotcache.BlockPairKey(userID, targetUserID) // 归一判定与 RelPairKey 同规则
	if a.cache != nil {
		if v, ok := a.cache.Get(ctx, key); ok {
			var cs cachedRelPair
			if json.Unmarshal([]byte(v), &cs) == nil {
				if swapped {
					return cs.FollowedBy, cs.Following, false, nil
				}
				return cs.Following, cs.FollowedBy, false, nil
			}
		}
	}
	relation, err := a.repo.Relationship(ctx, userID, targetUserID)
	if err != nil {
		return false, false, false, err
	}
	if a.cache != nil {
		first, second := relation.IsFollowing, relation.IsFollowedBy
		if swapped {
			first, second = relation.IsFollowedBy, relation.IsFollowing
		}
		if cs, err := json.Marshal(cachedRelPair{Following: first, FollowedBy: second}); err == nil {
			a.cache.Set(ctx, key, string(cs), hotcache.RelPairTTL)
		}
	}
	return relation.IsFollowing, relation.IsFollowedBy, relation.IsBlocked, err
}

func (a realtimeUserRepo) BlockedEitherWay(ctx context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	key, swapped := hotcache.BlockPairKey(aUserID, bUserID)
	if a.cache != nil {
		if v, ok := a.cache.Get(ctx, key); ok {
			mask := int(v[0] - '0')
			first, second := mask&1 == 1, mask&2 == 2
			if swapped {
				callerBlocked, targetBlocked = second, first
			} else {
				callerBlocked, targetBlocked = first, second
			}
			return callerBlocked, targetBlocked, nil
		}
	}
	callerBlocked, targetBlocked, err = a.repo.BlockedEitherWay(ctx, aUserID, bUserID)
	if err != nil {
		return false, false, err
	}
	if a.cache != nil {
		first, second := callerBlocked, targetBlocked
		if swapped {
			first, second = targetBlocked, callerBlocked
		}
		mask := 0
		if first {
			mask |= 1
		}
		if second {
			mask |= 2
		}
		a.cache.Set(ctx, key, string(rune('0'+mask)), hotcache.BlockPairTTL)
	}
	return callerBlocked, targetBlocked, nil
}

func (a realtimeGroupRepo) ListMembers(ctx context.Context, groupID string, limit int, cursor string) ([]string, string, error) {
	items, nextCursor, err := a.repo.ListMembers(ctx, groupID, nil, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	members := make([]string, 0, len(items))
	for _, item := range items {
		members = append(members, item.User.UserID)
	}
	return members, nextCursor, nil
}

func (a realtimeGroupRepo) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	key := hotcache.GroupMemberKey(groupID, userID)
	if a.cache != nil {
		if v, ok := a.cache.Get(ctx, key); ok {
			return v == "1", nil
		}
	}
	isMember, err := a.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return false, err
	}
	if a.cache != nil {
		if isMember {
			a.cache.Set(ctx, key, "1", hotcache.GroupMemberTTL)
		} else {
			a.cache.Set(ctx, key, "0", hotcache.GroupMemberTTL) // 防穿透
		}
	}
	return isMember, nil
}

// AreMembers 批量成员判定（realtime 订阅鉴权批量路径）。
func (a realtimeGroupRepo) AreMembers(ctx context.Context, groupIDs []string, userID string) (map[string]bool, error) {
	return a.repo.AreMembers(ctx, groupIDs, userID)
}

// reportsMediaSource adapts the media module Service to the reports snapshot
// source's file-metadata interface (keeps reports free of a media import).
type reportsMediaSource struct{ svc *media.Service }

func (a reportsMediaSource) CompletedFileMetaByID(ctx context.Context, fileID string) (reports.MediaFileMeta, bool, error) {
	meta, found, err := a.svc.CompletedFileMetaByID(ctx, fileID)
	if err != nil || !found {
		return reports.MediaFileMeta{}, found, err
	}
	return reports.MediaFileMeta{
		StorageKey:   meta.StorageKey,
		ThumbnailURL: meta.ThumbnailURL,
		Width:        meta.Width,
		Height:       meta.Height,
		MimeType:     meta.MimeType,
		FileName:     meta.FileName,
		FileSize:     meta.FileSize,
		Duration:     meta.Duration,
	}, true, nil
}
