package privatechats

import (
	"context"
	"sort"
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/types"
)

type userReader interface {
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
	// ChatTargetsContext 批量解析目标上下文（画像/在线态/双向关系/私聊权限），
	// 替代逐会话 OpenTheDoor + BlockStateBetween + ChatSettingsFor 的 N+1。
	ChatTargetsContext(context.Context, string, []string) (map[string]users.ChatTargetContext, error)
}

type Service struct {
	repo     Repository
	messages privatemessages.ReadService
	users    userReader
}

func NewServiceWithDeps(repo Repository, messages privatemessages.ReadService, userService userReader) *Service {
	return &Service{repo: repo, messages: messages, users: userService}
}
func (s *Service) OpenTheDoor(ctx context.Context, targetUserID string, recentLimit int) (PrivateChatOpenResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// 与 Delete/UpdateSettings 对齐：先确认目标用户存在，不存在按 10300
	// 处理，避免 GetOrCreate 在外键约束上失败并落到 10600。
	if _, ok, err := s.users.OpenTheDoor(ctx, targetUserID); err != nil {
		return PrivateChatOpenResult{}, err
	} else if !ok {
		return PrivateChatOpenResult{}, users.ErrUserNotFound
	}
	createdAt, err := s.repo.GetOrCreate(ctx, principal.UserID, targetUserID)
	if err != nil {
		observability.L().Error("privatechats.open_the_door get_or_create failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	state, err := s.repo.GetState(ctx, principal.UserID, targetUserID)
	if err != nil {
		observability.L().Error("privatechats.open_the_door get_state failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = createdAt
	}
	boundary := visibleBoundary(state)
	recent, err := s.messages.Recent(ctx, targetUserID, recentLimit, boundary)
	if err != nil {
		observability.L().Error("privatechats.open_the_door recent failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	if state.DeletedAt != nil && len(recent.Items) == 0 {
		now := time.Now().UTC()
		if err := s.repo.RestoreForUser(ctx, principal.UserID, targetUserID, now); err != nil {
			observability.L().Error("privatechats.open_the_door restore failed", "error", err, "target_user_id", targetUserID)
			return PrivateChatOpenResult{}, err
		}
		state.DeletedAt = nil
		state.VisibleSince = &now
	}
	chats, err := s.buildChats(ctx, principal.UserID, []ChatState{state})
	if err != nil {
		observability.L().Error("privatechats.open_the_door build_chat failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	if len(chats) == 0 {
		// 对方已注销/不存在（批量组装按不可达跳过）。
		return PrivateChatOpenResult{}, users.ErrUserNotFound
	}
	return PrivateChatOpenResult{Chat: chats[0], RecentMessages: recent}, nil
}
func (s *Service) List(ctx context.Context, req MyPrivateChatsRequest) (PaginatedPrivateChats, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	states, _, err := s.repo.List(ctx, principal.UserID, ListOptions{})
	if err != nil {
		observability.L().Error("privatechats.list failed", "error", err)
		return PaginatedPrivateChats{}, err
	}
	items, err := s.buildChats(ctx, principal.UserID, states)
	if err != nil {
		return PaginatedPrivateChats{}, err
	}
	// 排序键用解析后的时间值而非 RFC3339Nano 字符串：变长小数秒格式下
	// 字典序与时间序相反（"12:00:00Z" > "12:00:00.5Z"），同秒会话会错序。
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsPinned != items[j].IsPinned {
			return items[i].IsPinned
		}
		ati, _ := time.Parse(time.RFC3339Nano, items[i].LastActivityAt)
		atj, _ := time.Parse(time.RFC3339Nano, items[j].LastActivityAt)
		if !ati.Equal(atj) {
			return ati.After(atj)
		}
		return items[i].OtherUser.UserID < items[j].OtherUser.UserID
	})
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	cursorPinned, cursorAt, cursorUserID, err := decodeChatCursor(cursor)
	if err != nil {
		return PaginatedPrivateChats{}, err
	}
	if !cursorAt.IsZero() {
		filtered := make([]PrivateChatItem, 0, len(items))
		for _, item := range items {
			lastActivityAt, parseErr := time.Parse(time.RFC3339Nano, item.LastActivityAt)
			if parseErr != nil {
				continue
			}
			// 与排序键严格同序：置顶区整体在前，游标落在置顶区时仅当本条
			// 也置顶才参与时间比较，非置顶区同理，避免跨区过滤掉可达会话。
			if item.IsPinned != cursorPinned {
				if !item.IsPinned {
					filtered = append(filtered, item)
				}
				continue
			}
			if lastActivityAt.Before(cursorAt) || (lastActivityAt.Equal(cursorAt) && item.OtherUser.UserID > cursorUserID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	// 统一走 types.NormalizeLimit：spec api_spec_base.md 规定 limit default=20/max=100，
	// 此前直接采纳客户端值，limit=100000 会返回整份会话列表（超规格外泄 + 越过分页语义）。
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	if len(items) <= limit {
		return PaginatedPrivateChats{Items: items}, nil
	}
	next := items[limit-1]
	c := encodeChatCursor(next.IsPinned, next.LastActivityAt, next.OtherUser.UserID)
	return PaginatedPrivateChats{Items: items[:limit], NextCursor: &c}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, req UpdatePrivateChatSettingsRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// 与 Delete 对齐：先确认目标用户存在，不存在按 10300 处理，避免
	// GetOrCreate 在外键约束上失败并落到 10600。
	if _, ok, err := s.users.OpenTheDoor(ctx, req.TargetUserID); err != nil {
		return err
	} else if !ok {
		return users.ErrUserNotFound
	}
	if _, err := s.repo.GetOrCreate(ctx, principal.UserID, req.TargetUserID); err != nil {
		return err
	}
	if err := s.repo.UpdateSettings(ctx, principal.UserID, req.TargetUserID, UpdateChatSettings{IsMuted: req.IsMuted, IsPinned: req.IsPinned}); err != nil {
		return err
	}
	// 不推送 self-sync 事件（设置状态不做跨设备实时同步，见 api_spec_index.md 预留）
	return nil
}

func (s *Service) Delete(ctx context.Context, targetUserID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if _, ok, err := s.users.OpenTheDoor(ctx, targetUserID); err != nil {
		return err
	} else if !ok {
		return users.ErrUserNotFound
	}
	now := time.Now().UTC()
	return s.repo.DeleteForUser(ctx, principal.UserID, targetUserID, &now)
}

// buildChats 批量组装会话列表项（审计 perf H-1）：此前逐会话 buildChat 触发
// 5 类 N+1（users.OpenTheDoor 内部 ~7+ 查询、Last 恒读 21 行、UnreadCount、
// BlockStateBetween、ChatSettingsFor），50 会话 ≈ 500 次 DB 往返。现收敛为
// 3 次批量读（消息末条 DISTINCT ON、未读 GROUP BY、目标上下文批量），会话
// 数量级增长时往返数恒定。对方已注销的会话按不可达跳过（对齐 users 注销语义）。
func (s *Service) buildChats(ctx context.Context, userID string, states []ChatState) ([]PrivateChatItem, error) {
	items := make([]PrivateChatItem, 0, len(states))
	if len(states) == 0 {
		return items, nil
	}
	chatKeys := make([]string, 0, len(states))
	targetIDs := make([]string, 0, len(states))
	boundaries := make(map[string]*time.Time, len(states))
	for _, state := range states {
		chatKeys = append(chatKeys, state.ChatKey)
		targetIDs = append(targetIDs, state.TargetID)
		boundaries[state.ChatKey] = visibleBoundary(state)
	}
	lastByChat, err := s.messages.LastByChatKeys(ctx, chatKeys, boundaries)
	if err != nil {
		observability.L().Error("privatechats.build_chats last batch failed", "error", err)
		return nil, err
	}
	unreadByChat, err := s.messages.UnreadCountsByChatKeys(ctx, chatKeys, boundaries)
	if err != nil {
		observability.L().Error("privatechats.build_chats unread batch failed", "error", err)
		return nil, err
	}
	targets, err := s.users.ChatTargetsContext(ctx, userID, targetIDs)
	if err != nil {
		observability.L().Error("privatechats.build_chats targets batch failed", "error", err)
		return nil, err
	}
	for _, state := range states {
		target, ok := targets[state.TargetID]
		if !ok {
			// 对方已注销/不存在：跳过该会话而非让整个列表失败
			//（spec api_spec_user.md：注销用户不再可被访问）。
			continue
		}
		var last *privatemessages.PrivateMessageItem
		if item, found := lastByChat[state.ChatKey]; found {
			last = &item
		}
		item := s.assembleChat(userID, state, target, last, unreadByChat[state.ChatKey])
		if state.DeletedAt != nil && item.LastMessage == nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// assembleChat 由批量取回的数据纯内存组装单个会话项，权限判定与
// can_message/blocked_reason 口径见行内注释（契约 api_spec_chat.md）。
func (s *Service) assembleChat(userID string, state ChatState, target users.ChatTargetContext, last *privatemessages.PrivateMessageItem, unreadCount int) PrivateChatItem {
	item := PrivateChatItem{
		OtherUser:      UserInfo{UserID: target.Info.UserID, Nickname: target.Info.Nickname, AvatarURL: target.Info.AvatarURL},
		UnreadCount:    unreadCount,
		IsMuted:        state.IsMuted,
		IsPinned:       state.IsPinned,
		LastActivityAt: state.CreatedAt.UTC().Format(time.RFC3339Nano),
		CreatedAt:      func() *string { c := state.CreatedAt.UTC().Format(time.RFC3339Nano); return &c }(),
		CanMessage:     true,
	}
	if target.PresenceStatus != nil {
		ps := string(*target.PresenceStatus)
		item.PresenceStatus = &ps
	}
	// 契约(api_spec_chat.md → PrivateChatItem)：blocked_reason 仅 can_message=false
	// 时返回 blocked_by_user / privacy_setting / stranger_quota_exceeded。
	blockReason := func(reason httpx.ForbiddenReason) {
		r := reason
		item.CanMessage = false
		item.ForbiddenReason = &r
	}
	// 双向黑名单都要置 can_message=false：Relation.IsBlocked 是查看者视角
	//（查看者拉黑了对方），对方拉黑查看者看 TargetBlocked。
	if target.Relation.IsBlocked || target.Relation.TargetBlocked {
		blockReason(httpx.ReasonBlockedByUser)
	}
	// 对方的私聊权限设置：friends-only 需双向关注。
	if target.SettingsFound {
		switch target.AllowPrivateChat {
		case users.ChatPermNobody:
			blockReason(httpx.ReasonPrivacySetting)
		case users.ChatPermFriends:
			if !(target.Relation.IsFollowing && target.Relation.IsFollowedBy) {
				blockReason(httpx.ReasonPrivacySetting)
			}
		}
	}
	if userID == state.TargetID {
		item.CanMessage = true
		item.ForbiddenReason = nil
	}
	if last != nil {
		item.LastActivityAt = last.SentAt
		item.LastMessage = &LastMessage{MessageID: last.MessageID, Content: last.Content, Type: last.Type, ShareCode: last.ShareCode, Sender: UserInfo{UserID: last.Sender.UserID, Nickname: last.Sender.Nickname, AvatarURL: last.Sender.AvatarURL}, SentAt: last.SentAt}
	}
	return item
}

func visibleBoundary(state ChatState) *time.Time {
	if state.VisibleSince != nil {
		return state.VisibleSince
	}
	return state.DeletedAt
}
