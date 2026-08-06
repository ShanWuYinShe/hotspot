package users

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

func (s *Service) Follow(ctx context.Context, targetUserID string, source FollowSource, sourceMomentID string) error {
	startedAt := time.Now()
	observability.L().Info("users.follow started", "target_user_id", targetUserID)
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
	if err := s.repo.Follow(ctx, principal.UserID, targetUserID, source, sourceMomentID); err != nil {
		observability.L().Error("users.follow failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
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
			observability.L().Error("users.follow notification append failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
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
		observability.L().Error("users.follow relationship check failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if relation.IsFollowing {
		mutualAt := time.Now().UTC().Format(time.RFC3339)
		notificationIDForPrincipal := ""
		notificationIDForTarget := ""
		if s.notifications != nil {
			id, err := s.notifications.AppendMutualFollow(ctx, principal.UserID, actor)
			if err != nil {
				observability.L().Error("users.follow mutual notification append failed", "error", err, "target_user_id", principal.UserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
			} else {
				notificationIDForPrincipal = id
			}
			id, err = s.notifications.AppendMutualFollow(ctx, targetUserID, actor)
			if err != nil {
				observability.L().Error("users.follow mutual notification append failed", "error", err, "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
			} else {
				notificationIDForTarget = id
			}
		}
		mutualPayloadForPrincipal := map[string]any{"notify_id": notificationIDForPrincipal, "actor": actor, "mutual_at": mutualAt}
		mutualPayloadForTarget := map[string]any{"notify_id": notificationIDForTarget, "actor": actor, "mutual_at": mutualAt}
		realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeMutualFollow, Payload: mutualPayloadForPrincipal})
		realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeMutualFollow, Payload: mutualPayloadForTarget})
	}
	observability.L().Info("users.follow completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Unfollow(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.unfollow started", "target_user_id", targetUserID)
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
		observability.L().Error("users.unfollow failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeSelfRelationSync, Payload: map[string]any{
		"action":         "unfollow",
		"target_user_id": targetUserID,
	}})
	observability.L().Info("users.unfollow completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Block(ctx context.Context, targetUserID, reason string) error {
	startedAt := time.Now()
	observability.L().Info("users.block started", "target_user_id", targetUserID)
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
	if err := s.repo.Unfollow(ctx, principal.UserID, targetUserID); err != nil {
		observability.L().Error("users.block unfollow self failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if err := s.repo.Unfollow(ctx, targetUserID, principal.UserID); err != nil {
		observability.L().Error("users.block unfollow reverse failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if err := s.repo.Block(ctx, principal.UserID, targetUserID, reason); err != nil {
		observability.L().Error("users.block failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeSelfRelationSync, Payload: map[string]any{
		"action":         "block",
		"target_user_id": targetUserID,
	}})
	observability.L().Info("users.block completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Unblock(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.unblock started", "target_user_id", targetUserID)
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
		observability.L().Error("users.unblock failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeSelfRelationSync, Payload: map[string]any{
		"action":         "unblock",
		"target_user_id": targetUserID,
	}})
	observability.L().Info("users.unblock completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) SetRemark(ctx context.Context, targetUserID, remark string) error {
	startedAt := time.Now()
	observability.L().Info("users.set_remark started", "target_user_id", targetUserID)
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
		observability.L().Error("users.set_remark failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.set_remark completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) ListBlocked(ctx context.Context, req MyBlockedRequest) (PaginatedBlockedUsers, error) {
	startedAt := time.Now()
	observability.L().Info("users.list_blocked started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return PaginatedBlockedUsers{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	items, err := s.repo.ListBlocked(ctx, principal.UserID, limit, cursor)
	if err != nil {
		observability.L().Error("users.list_blocked failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedBlockedUsers{}, err
	}
	observability.L().Info("users.list_blocked completed", "count", len(items.Items), "elapsed_ms", time.Since(startedAt).Milliseconds())
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
	mutualFriends, err := collectMutualFriends(ctx, s.repo, viewerUserID, targetProfile.UserID)
	if err != nil {
		return nil, err
	}
	context := &SocialContext{MutualFriendsCount: len(mutualFriends)}
	if len(mutualFriends) > 0 {
		preview := make([]UserInfo, 0, minInt(len(mutualFriends), 3))
		for _, friend := range mutualFriends {
			preview = append(preview, friend.User)
			if len(preview) == 3 {
				break
			}
		}
		context.MutualFriendsPreview = preview
	}
	targetSettings, ok, err := s.repo.GetSettings(ctx, targetProfile.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		targetSettings = defaultSettings()
	}
	if targetSettings.GroupsVisible {
		commonGroupsCount, err := s.repo.CommonGroupCount(ctx, viewerUserID, targetProfile.UserID)
		if err != nil {
			return nil, err
		}
		context.CommonGroupsCount = &commonGroupsCount
	}
	commonTags := intersectTags(viewerProfile.Tags, targetProfile.Tags)
	if len(commonTags) > 0 {
		context.CommonTags = commonTags
	}
	return context, nil
}

func collectMutualFriends(ctx context.Context, repo Repository, viewerUserID, targetUserID string) ([]FriendItem, error) {
	// TODO(perf): this still paginates each user's entire friend list in memory and
	// intersects them in Go. It should be replaced by a single SQL-based mutual-friends
	// query (self-join on user_relations) that returns the intersection directly. Until
	// that lands, mutualFriendsScanCap bounds the number of friends scanned per user so
	// very large friend lists cannot blow up into unbounded pagination.
	viewerFriends, err := collectFriendsByUser(ctx, repo, viewerUserID)
	if err != nil {
		return nil, err
	}
	targetFriends, err := collectFriendsByUser(ctx, repo, targetUserID)
	if err != nil {
		return nil, err
	}
	targetSet := make(map[string]struct{}, len(targetFriends))
	for _, item := range targetFriends {
		targetSet[item.User.UserID] = struct{}{}
	}
	mutual := make([]FriendItem, 0)
	for _, item := range viewerFriends {
		if _, ok := targetSet[item.User.UserID]; ok {
			mutual = append(mutual, item)
		}
	}
	return mutual, nil
}

// mutualFriendsScanCap is a temporary upper bound on how many friend rows are
// scanned per user when computing mutual friends. See collectMutualFriends.
const mutualFriendsScanCap = 500

func collectFriendsByUser(ctx context.Context, repo Repository, userID string) ([]FriendItem, error) {
	items := make([]FriendItem, 0)
	cursor := ""
	pageSize := 100
	for {
		page, err := repo.ListFriends(ctx, userID, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		// Stop once we've scanned the cap or there are no more pages.
		if page.NextCursor == "" || len(items) >= mutualFriendsScanCap {
			if len(items) > mutualFriendsScanCap {
				items = items[:mutualFriendsScanCap]
			}
			break
		}
		cursor = page.NextCursor
	}
	return items, nil
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

func minInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func applyPresenceVisibilityToFriends(ctx context.Context, repo Repository, items []FriendItem) {
	hidden := hiddenPresenceUserIDs(ctx, repo, len(items), func(i int) string { return items[i].User.UserID })
	for i := range items {
		if hidden[items[i].User.UserID] {
			items[i].PresenceStatus = nil
			items[i].LastOnline = nil
		}
	}
}

func applyPresenceVisibilityToFollowing(ctx context.Context, repo Repository, items []FollowingItem) {
	hidden := hiddenPresenceUserIDs(ctx, repo, len(items), func(i int) string { return items[i].User.UserID })
	for i := range items {
		if hidden[items[i].User.UserID] {
			items[i].PresenceStatus = nil
			items[i].LastOnline = nil
		}
	}
}

func applyPresenceVisibilityToFollowers(ctx context.Context, repo Repository, items []FollowerItem) {
	hidden := hiddenPresenceUserIDs(ctx, repo, len(items), func(i int) string { return items[i].User.UserID })
	for i := range items {
		if hidden[items[i].User.UserID] {
			items[i].PresenceStatus = nil
			items[i].LastOnline = nil
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
	return types.UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: profile.AvatarURL}
}
