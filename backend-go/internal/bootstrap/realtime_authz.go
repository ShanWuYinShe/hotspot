package bootstrap

import (
	"context"
	"time"

	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/actor"
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

type realtimeUserRepo struct{ repo users.Repository }

type realtimeGroupRepo struct{ repo groups.Repository }

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
	settings, ok, err := a.repo.GetSettings(ctx, userID)
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

func (a realtimeUserRepo) Relationship(ctx context.Context, userID, targetUserID string) (bool, bool, bool, error) {
	relation, err := a.repo.Relationship(ctx, userID, targetUserID)
	return relation.IsFollowing, relation.IsFollowedBy, relation.IsBlocked, err
}

func (a realtimeUserRepo) BlockedEitherWay(ctx context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	return a.repo.BlockedEitherWay(ctx, aUserID, bUserID)
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
	return a.repo.IsMember(ctx, groupID, userID)
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
