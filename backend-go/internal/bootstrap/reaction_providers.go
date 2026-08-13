package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/reactions"
	"hotspot/backend-go/internal/platform/types"
)

// messageExistence is the minimal surface the message reaction provider needs
// from the chat services: existence + 目标可见性（spec reaction.md 目标可见性）。
type messageExistence interface {
	LookupMessage(ctx context.Context, messageID string) (found bool, senderID string, err error)
	IsReactionTargetVisible(ctx context.Context, messageID string) (found bool, visible bool, err error)
}

// momentReactionProvider adapts moments.Service into a reactions.TargetProvider
// for the 'moment' target kind. It reuses the existing Exists /
// NotificationContext surface so reactions never reaches into the moments repo
// directly (keeping the dependency one-way: reactions -> moments.Service only).
type momentReactionProvider struct{ moments *moments.Service }

func (p momentReactionProvider) ReactTarget() reactions.ReactTarget {
	return reactions.ReactTargetMoment
}

func (p momentReactionProvider) Resolve(ctx context.Context, momentID string) (reactions.TargetContext, bool, error) {
	if p.moments == nil {
		return reactions.TargetContext{}, true, nil
	}
	// 目标可见性（spec reaction.md）：存在但不可见（private 非拥有者）按不存在处理
	found, visible, err := p.moments.IsReactionTargetVisible(ctx, momentID)
	if err != nil {
		return reactions.TargetContext{}, false, err
	}
	if !found || !visible {
		return reactions.TargetContext{}, false, nil
	}
	mc, ok, err := p.moments.NotificationContext(ctx, momentID)
	if err != nil {
		return reactions.TargetContext{}, false, err
	}
	if !ok {
		return reactions.TargetContext{}, false, nil
	}
	summary := mc.Summary
	if summary.MomentID == "" {
		summary.MomentID = momentID
	}
	return reactions.TargetContext{OwnerUserID: mc.UserID, Summary: summary}, true, nil
}

// commentReactionProvider adapts comments.Service into a reactions.TargetProvider
// for the 'comment' target kind. moments 引用用于校验评论挂载瞬间的可见性
// （spec reaction.md 目标可见性：comment → 所属瞬间）。
type commentReactionProvider struct {
	comments *comments.Service
	moments  *moments.Service
}

func (p commentReactionProvider) ReactTarget() reactions.ReactTarget {
	return reactions.ReactTargetComment
}

func (p commentReactionProvider) Resolve(ctx context.Context, commentID string) (reactions.TargetContext, bool, error) {
	if p.comments == nil {
		return reactions.TargetContext{}, true, nil
	}
	item, found, err := p.comments.Get(ctx, commentID)
	if err != nil {
		return reactions.TargetContext{}, false, err
	}
	if !found {
		return reactions.TargetContext{}, false, nil
	}
	// 目标可见性（spec reaction.md）：评论挂载的瞬间不可见（private 非拥有者）时按不存在处理
	if p.moments != nil {
		if _, visible, err := p.moments.IsReactionTargetVisible(ctx, item.MomentID); err != nil {
			return reactions.TargetContext{}, false, err
		} else if !visible {
			return reactions.TargetContext{}, false, nil
		}
	}
	// reaction_added 通知/事件载荷不含 summary 字段（spec），仅保留
	// moment_id 便于关联定位。
	return reactions.TargetContext{OwnerUserID: item.User.UserID, Summary: types.MomentSummary{MomentID: item.MomentID}}, true, nil
}

// messageReactionProvider resolves message existence for the 'message'
// reaction target kind. Owner notification for message reactions is handled
// by the chat layer (see reactions/deps.go TargetContext doc), so we return
// an empty OwnerUserID. The provider probes both group and private message
// backends; a hit on either means the target exists.
type messageReactionProvider struct {
	groupMessages   messageExistence
	privateMessages messageExistence
}

func (p messageReactionProvider) ReactTarget() reactions.ReactTarget {
	return reactions.ReactTargetMessage
}

func (p messageReactionProvider) Resolve(ctx context.Context, messageID string) (reactions.TargetContext, bool, error) {
	// 目标可见性（spec reaction.md）：私聊仅参与者、群聊仅群成员可见；
	// 存在但不可见（被移除/退群/非参与者）按不存在处理。
	if p.groupMessages != nil {
		found, visible, err := p.groupMessages.IsReactionTargetVisible(ctx, messageID)
		if err != nil {
			return reactions.TargetContext{}, false, err
		}
		if found {
			return reactions.TargetContext{}, visible, nil
		}
	}
	if p.privateMessages != nil {
		found, visible, err := p.privateMessages.IsReactionTargetVisible(ctx, messageID)
		if err != nil {
			return reactions.TargetContext{}, false, err
		}
		if found {
			return reactions.TargetContext{}, visible, nil
		}
	}
	return reactions.TargetContext{}, false, nil
}
