package users

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/hotcache"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// invalidateRelationCache 关注/拉黑变更后双向失效两对键（读侧为 realtime
// typing 每帧判定，漏删会让新关系在 TTL 内不生效）。
func (s *Service) invalidateRelationCache(ctx context.Context, aUserID, bUserID string) {
	if s == nil || s.settingsCache == nil {
		return
	}
	blkKey, _ := hotcache.BlockPairKey(aUserID, bUserID)
	s.settingsCache.Delete(ctx, blkKey, hotcache.RelPairKey(aUserID, bUserID))
}

func (s *Service) Follow(ctx context.Context, targetUserID string, source FollowSource, sourceMomentID string) error {
	op := observability.Begin("users.follow", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	settings, ok, err := s.repo.GetSettings(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		settings = defaultSettings()
	}
	if !settings.AllowFollow {
		return ErrPermissionDenied
	}
	// changed=false 表示此前已关注（no-op）：跳过全部通知与实时事件广播。
	// spec api_spec_user.md set_follow：重复关注同一状态应为 no-op（既不重复
	// 落通知也不重复推送）——此前 upsert 恒成功，反复 follow/unfollow 每轮都
	// 会再发一条 new_follower（并可能重复 mutual_follow）（审计 F-12）。
	changed, err := s.repo.Follow(ctx, principal.UserID, targetUserID, source, sourceMomentID)
	if err != nil {
		op.Fail(err)
		return err
	}
	if !changed {
		observability.L().Info("users.follow no-op (already following)", "target_user_id", targetUserID, "elapsed_ms", op.ElapsedMillis())
		return nil
	}
	actor := s.buildFollowActor(ctx, principal.UserID)
	followedAt := time.Now().UTC().Format(time.RFC3339)
	sourceValue := strings.TrimSpace(string(source))
	sourceMomentID = strings.TrimSpace(sourceMomentID)
	if sourceValue != string(FollowSourceMoment) {
		sourceMomentID = ""
	}
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendNewFollower(ctx, targetUserID, actor, sourceValue, sourceMomentID)
		if err != nil {
			observability.L().Error("users.follow notification append failed", "error", err, "elapsed_ms", op.ElapsedMillis())
		} else {
			notificationID = id
		}
	}
	newFollowerPayload := map[string]any{"notify_id": notificationID, "actor": actor, "followed_at": followedAt}
	if sourceValue != "" {
		newFollowerPayload["source"] = sourceValue
	}
	if sourceMomentID != "" {
		newFollowerPayload["source_moment_id"] = sourceMomentID
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeNewFollower, Payload: newFollowerPayload})
	relation, err := s.repo.Relationship(ctx, targetUserID, principal.UserID)
	if err != nil {
		op.Fail(err, "stage", "relationship-check")
		return err
	}
	if relation.IsFollowing {
		mutualAt := time.Now().UTC().Format(time.RFC3339)
		notificationIDForPrincipal := ""
		notificationIDForTarget := ""
		if s.notifications != nil {
			id, err := s.notifications.AppendMutualFollow(ctx, principal.UserID, actor)
			if err != nil {
				observability.L().Error("users.follow mutual notification append failed", "error", err, "target_user_id", principal.UserID, "elapsed_ms", op.ElapsedMillis())
			} else {
				notificationIDForPrincipal = id
			}
			id, err = s.notifications.AppendMutualFollow(ctx, targetUserID, actor)
			if err != nil {
				observability.L().Error("users.follow mutual notification append failed", "error", err, "target_user_id", targetUserID, "elapsed_ms", op.ElapsedMillis())
			} else {
				notificationIDForTarget = id
			}
		}
		mutualPayloadForPrincipal := map[string]any{"notify_id": notificationIDForPrincipal, "actor": actor, "mutual_at": mutualAt}
		mutualPayloadForTarget := map[string]any{"notify_id": notificationIDForTarget, "actor": actor, "mutual_at": mutualAt}
		realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeMutualFollow, Payload: mutualPayloadForPrincipal})
		realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeMutualFollow, Payload: mutualPayloadForTarget})
	}
	s.invalidateRelationCache(ctx, principal.UserID, targetUserID)
	op.Complete()
	return nil
}

func (s *Service) Unfollow(ctx context.Context, targetUserID string) error {
	op := observability.Begin("users.unfollow", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	if err := s.repo.Unfollow(ctx, principal.UserID, targetUserID); err != nil {
		op.Fail(err)
		return err
	}
	s.invalidateRelationCache(ctx, principal.UserID, targetUserID)
	// 不推送 self-sync 事件（关系状态不做跨设备实时同步，见 api_spec_index.md 预留）
	op.Complete()
	return nil
}

func (s *Service) Block(ctx context.Context, targetUserID, reason string) error {
	op := observability.Begin("users.block", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	// 级联语义（spec api_spec_user.md block）：拉黑自动解除双方关注/好友关系，
	// 解除拉黑不自动恢复（需重新关注）。与微博/Instagram 等主流产品一致。
	// BlockCascade 在同一事务内完成三步写，避免中间状态。
	if err := s.repo.BlockCascade(ctx, principal.UserID, targetUserID, reason); err != nil {
		op.Fail(err)
		return err
	}
	// 拉黑级联解除双方关注：关系与拉黑两对键都失效。
	s.invalidateRelationCache(ctx, principal.UserID, targetUserID)
	// 不推送 self-sync 事件（关系状态不做跨设备实时同步，见 api_spec_index.md 预留）
	op.Complete()
	return nil
}

// BlockStateBetween 返回 a、b 之间的双向拉黑状态：callerBlocked = a 拉黑了 b，
// targetBlocked = b 拉黑了 a。供私聊发送/typing 等场景做双向黑名单拦截——
// Relationship 的 IsBlocked 只覆盖正向（第一参数拉黑第二参数），直接用会把
// "被对方拉黑"误判成"我拉黑了对方"（spec api_spec_chat.md：send 必须校验黑名单）。
func (s *Service) BlockStateBetween(ctx context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	return s.repo.BlockedEitherWay(ctx, aUserID, bUserID)
}

// ChatTargetContext 会话列表单目标上下文（ChatTargetsContext 的批量产物，
// 供 privatechats.List 一次组装，替代逐会话 OpenTheDoor 的 7+ 查询 N+1）。
type ChatTargetContext struct {
	// Info 基础画像（id/昵称/头像）。
	Info UserInfo
	// PresenceStatus 经对方 presence_visible 门控后的在线状态（nil = 不可见）。
	PresenceStatus *UserPresenceStatus
	// Relation viewer→target 的双向关系快照：IsFollowing/IsFollowedBy 供
	// friends-only 私聊权限判定，IsBlocked/TargetBlocked 供双向拉黑拦截
	//（与单查 Relationship 口径一致；恒非 nil）。
	Relation Relationship
	// AllowPrivateChat 对方私聊权限设置；SettingsFound=false 时未持久化
	//（调用方按默认 everyone 处理）。
	AllowPrivateChat ChatPerm
	SettingsFound    bool
}

// ChatTargetsContext 批量解析 viewer 对多个目标的会话上下文：3 次批量往返
// （画像、设置、关系双向边）替代逐目标 OpenTheDoor + BlockStateBetween +
// ChatSettingsFor 的 N+1。已注销/不存在目标不出现在结果中。
func (s *Service) ChatTargetsContext(ctx context.Context, viewerID string, targetUserIDs []string) (map[string]ChatTargetContext, error) {
	if len(targetUserIDs) == 0 {
		return map[string]ChatTargetContext{}, nil
	}
	profiles, err := s.repo.GetProfiles(ctx, targetUserIDs)
	if err != nil {
		return nil, err
	}
	live := make([]string, 0, len(targetUserIDs))
	for _, id := range targetUserIDs {
		if profile, ok := profiles[id]; ok && !profile.IsDeleted {
			live = append(live, id)
		}
	}
	settingsBatch, err := s.repo.GetSettingsBatch(ctx, live)
	if err != nil {
		return nil, err
	}
	relations, err := s.repo.RelationshipBatch(ctx, viewerID, live)
	if err != nil {
		return nil, err
	}
	result := make(map[string]ChatTargetContext, len(live))
	for _, id := range live {
		profile := profiles[id]
		settings, found := settingsBatch[id]
		presenceVisible := !found || settings.PresenceVisible
		item := ChatTargetContext{
			Info:             UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: types.StrPtr(profile.AvatarURL)},
			Relation:         relations[id],
			AllowPrivateChat: settings.AllowPrivateChat,
			SettingsFound:    found,
		}
		if presenceVisible {
			ps := profile.PresenceStatus
			item.PresenceStatus = &ps
		}
		result[id] = item
	}
	return result, nil
}

func (s *Service) Unblock(ctx context.Context, targetUserID string) error {
	op := observability.Begin("users.unblock", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	if err := s.repo.Unblock(ctx, principal.UserID, targetUserID); err != nil {
		op.Fail(err)
		return err
	}
	s.invalidateRelationCache(ctx, principal.UserID, targetUserID)
	// 不推送 self-sync 事件（关系状态不做跨设备实时同步，见 api_spec_index.md 预留）
	op.Complete()
	return nil
}

func (s *Service) SetRemark(ctx context.Context, targetUserID, remark string) error {
	op := observability.Begin("users.set_remark", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	relation, err := s.repo.Relationship(ctx, principal.UserID, targetUserID)
	if err != nil {
		return err
	}
	if !relation.IsFollowing {
		return ErrConflict
	}
	if err := s.repo.SetRemark(ctx, principal.UserID, targetUserID, remark); err != nil {
		op.Fail(err)
		return err
	}
	op.Complete()
	return nil
}

func (s *Service) ListBlocked(ctx context.Context, req MyBlockedRequest) (PaginatedBlockedUsers, error) {
	op := observability.Begin("users.list_blocked")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return PaginatedBlockedUsers{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	items, err := s.repo.ListBlocked(ctx, principal.UserID, limit, cursor)
	if err != nil {
		op.Fail(err)
		return PaginatedBlockedUsers{}, err
	}
	op.Complete("count", len(items.Items))
	return items, nil
}

func (s *Service) buildSocialContext(ctx context.Context, viewerUserID string, targetProfile MyProfile) (*SocialContext, error) {
	viewerProfile, ok, err := s.repo.GetProfile(ctx, viewerUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		viewerProfile = s.defaultProfile(ctx, viewerUserID)
	}
	mutualTotal, mutualPreview, err := s.repo.MutualFriends(ctx, viewerUserID, targetProfile.UserID, 3)
	if err != nil {
		return nil, err
	}
	context := &SocialContext{MutualFriendsCount: mutualTotal}
	if len(mutualPreview) > 0 {
		context.MutualFriendsPreview = mutualPreview
	}
	targetSettings, ok, err := s.repo.GetSettings(ctx, targetProfile.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		targetSettings = defaultSettings()
	}
	if targetSettings.GroupsVisible && s.groupsByUser != nil {
		commonGroupsCount, err := s.groupsByUser.CountCommonGroups(ctx, viewerUserID, targetProfile.UserID)
		if err != nil {
			return nil, err
		}
		context.CommonGroupsCount = &commonGroupsCount
	} else if targetSettings.GroupsVisible {
		observability.L().Warn("users.social_context groups provider not wired")
	}
	commonTags := intersectTags(viewerProfile.Tags, targetProfile.Tags)
	if len(commonTags) > 0 {
		context.CommonTags = commonTags
	}
	// 无任何共同点 → null（恒定 nullable：null = 不适用，spec api_spec_base.md）。
	// 不返回 {"mutual_friends_count":0} 这类空壳对象。
	if mutualTotal == 0 && len(commonTags) == 0 && (context.CommonGroupsCount == nil || *context.CommonGroupsCount == 0) {
		return nil, nil
	}
	return context, nil
}

func intersectTags(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	seen := make(map[string]string, len(left))
	for _, item := range left {
		key := strings.ToLower(item)
		if _, ok := seen[key]; !ok {
			seen[key] = item
		}
	}
	common := make([]string, 0)
	added := make(map[string]struct{})
	for _, item := range right {
		key := strings.ToLower(item)
		original, ok := seen[key]
		if !ok {
			continue
		}
		if _, dup := added[key]; dup {
			continue
		}
		common = append(common, original)
		added[key] = struct{}{}
	}
	return common
}

// applyPresenceVisibility 按 presence_visible 隐藏列表项的在线状态
// （审查 P2-25：Friends/Following/Followers 三个列表原是三份逐字相同的
// 实现，收敛为一个；行为不变——设置读取失败或无设置行时保持可见）。
func applyPresenceVisibility(ctx context.Context, repo Repository, items []RelatedUserItem) {
	hidden := hiddenPresenceUserIDs(ctx, repo, len(items), func(i int) string { return items[i].User.User.UserID })
	for i := range items {
		if hidden[items[i].User.User.UserID] {
			items[i].User.PresenceStatus = nil
			items[i].User.LastOnline = nil
		}
	}
}

// hiddenPresenceUserIDs resolves each item's user ID against a single
// GetSettingsBatch call and returns the set of IDs whose presence must be
// hidden. On settings-read error or when a user has no stored settings, the
// previous per-item behavior is preserved: presence stays visible (not hidden).
func hiddenPresenceUserIDs(ctx context.Context, repo Repository, count int, userIDAt func(i int) string) map[string]bool {
	hidden := make(map[string]bool, count)
	if count == 0 {
		return hidden
	}
	userIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		userIDs = append(userIDs, userIDAt(i))
	}
	settingsByUserID, err := repo.GetSettingsBatch(ctx, userIDs)
	if err != nil {
		return hidden
	}
	for _, userID := range userIDs {
		settings, ok := settingsByUserID[userID]
		if !ok {
			settings = defaultSettings()
		}
		if !settings.PresenceVisible {
			hidden[userID] = true
		}
	}
	return hidden
}

func ensureSocialListsVisible(ctx context.Context, repo Repository, userID string, isSelf bool) error {
	if isSelf {
		return nil
	}
	settings, ok, err := repo.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		settings = defaultSettings()
	}
	if !settings.SocialListsVisible {
		return ErrPermissionDenied
	}
	return nil
}

func (s *Service) buildFollowActor(ctx context.Context, userID string) types.UserInfo {
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil || !ok {
		return types.UserInfo{UserID: userID, Nickname: userID}
	}
	return types.UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: types.StrPtr(profile.AvatarURL)}
}
