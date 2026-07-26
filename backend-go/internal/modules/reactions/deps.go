package reactions

import "context"

// TargetContext carries the author and display summary of a reaction target,
// used to address notifications/realtime when a reaction is added.
type TargetContext struct {
	// OwnerUserID is the author of the target resource (moment/comment).
	// Empty for message targets whose owner is resolved per-message by the
	// chat layer; reactions to such targets skip owner notification.
	OwnerUserID string
	// Summary is an opaque payload describing the target for notification
	// rendering (e.g. moment_id + description preview).
	Summary map[string]any
}

// TargetProvider resolves a target's context (existence + owner + summary)
// for one reaction target kind. Each provider handles a single ReactTarget;
// the reactions Service composes them per call.
type TargetProvider interface {
	// ReactTarget returns the kind this provider serves.
	ReactTarget() ReactTarget
	// Resolve returns the target context. found=false means the target does
	// not exist (caller maps to ErrReactionTargetNotFound).
	Resolve(ctx context.Context, targetID string) (ctx2 TargetContext, found bool, err error)
}

// momentNotificationWriter is the narrow notifications surface reactions
// needs. Defined locally (like comments/momentNotificationWriter) so the
// reactions package depends on an interface, not the notifications package's
// concrete Service.
type reactionNotificationWriter interface {
	AppendMomentLiked(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any) error
	AppendCommentLiked(ctx context.Context, targetUserID string, actor map[string]any, commentID string, commentPreview string, momentID string, momentSummary map[string]any) error
	AppendReactionAdded(ctx context.Context, targetUserID string, actor map[string]any, targetType string, targetID string, emoji string) error
}
