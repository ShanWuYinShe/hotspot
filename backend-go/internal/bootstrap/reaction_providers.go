package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/reactions"
)

// messageExistence is the minimal surface the message reaction provider needs
// from the chat services — just determine whether a message exists by its ID.
type messageExistence interface {
	LookupMessage(ctx context.Context, messageID string) (found bool, senderID string, err error)
}

// momentReactionProvider adapts moments.Service into a reactions.TargetProvider
// for the 'moment' target kind. It reuses the existing Exists /
// NotificationContext surface so reactions never reaches into the moments repo
// directly (keeping the dependency one-way: reactions -> moments.Service only).
type momentReactionProvider struct{ moments *moments.Service }

func (p momentReactionProvider) ReactTarget() reactions.ReactTarget { return reactions.ReactTargetMoment }

func (p momentReactionProvider) Resolve(ctx context.Context, momentID string) (reactions.TargetContext, bool, error) {
	if p.moments == nil {
		return reactions.TargetContext{}, true, nil
	}
	exists, err := p.moments.Exists(ctx, momentID)
	if err != nil {
		return reactions.TargetContext{}, false, err
	}
	if !exists {
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
	if summary == nil {
		summary = map[string]any{"moment_id": momentID}
	}
	return reactions.TargetContext{OwnerUserID: mc.UserID, Summary: summary}, true, nil
}

// commentReactionProvider adapts comments.Service into a reactions.TargetProvider
// for the 'comment' target kind.
type commentReactionProvider struct{ comments *comments.Service }

func (p commentReactionProvider) ReactTarget() reactions.ReactTarget { return reactions.ReactTargetComment }

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
	summary := map[string]any{
		"moment_id":       item.MomentID,
		"comment_preview": summarizeCommentPreview(item.Content),
	}
	return reactions.TargetContext{OwnerUserID: item.User.UserID, Summary: summary}, true, nil
}

// summarizeCommentPreview mirrors comments' own preview helper so the
// reaction notification carries a short text without importing comments'
// private helpers.
func summarizeCommentPreview(content string) string {
	const max = 80
	if len(content) > max {
		return content[:max]
	}
	return content
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

func (p messageReactionProvider) ReactTarget() reactions.ReactTarget { return reactions.ReactTargetMessage }

func (p messageReactionProvider) Resolve(ctx context.Context, messageID string) (reactions.TargetContext, bool, error) {
	if p.groupMessages != nil {
		found, _, err := p.groupMessages.LookupMessage(ctx, messageID)
		if err != nil {
			return reactions.TargetContext{}, false, err
		}
		if found {
			return reactions.TargetContext{}, true, nil
		}
	}
	if p.privateMessages != nil {
		found, _, err := p.privateMessages.LookupMessage(ctx, messageID)
		if err != nil {
			return reactions.TargetContext{}, false, err
		}
		if found {
			return reactions.TargetContext{}, true, nil
		}
	}
	return reactions.TargetContext{}, false, nil
}
