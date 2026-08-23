package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/reactions"
)

// momentReactionAggregator adapts reactions.Service to the
// moments.ReactionAggregator interface. It performs the trivial struct copy
// between reactions.Reaction and moments.Reaction so the two modules never
// import each other (reactions stays a leaf dependency).
type momentReactionAggregator struct{ reactions *reactions.Service }

func (a momentReactionAggregator) AggregateMoments(ctx context.Context, momentIDs []string, viewerID string) (map[string][]moments.Reaction, error) {
	src, err := a.reactions.AggregateBatch(ctx, reactions.ReactTargetMoment, momentIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]moments.Reaction, len(src))
	for id, recs := range src {
		conv := make([]moments.Reaction, len(recs))
		for i, r := range recs {
			conv[i] = moments.Reaction{Emoji: r.Emoji, Count: r.Count, ReactedByMe: r.ReactedByMe}
		}
		out[id] = conv
	}
	return out, nil
}

// commentReactionAggregator adapts reactions.Service to the
// comments.ReactionAggregator interface.
type commentReactionAggregator struct{ reactions *reactions.Service }

func (a commentReactionAggregator) AggregateComments(ctx context.Context, commentIDs []string, viewerID string) (map[string][]comments.Reaction, error) {
	src, err := a.reactions.AggregateBatch(ctx, reactions.ReactTargetComment, commentIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]comments.Reaction, len(src))
	for id, recs := range src {
		conv := make([]comments.Reaction, len(recs))
		for i, r := range recs {
			conv[i] = comments.Reaction{Emoji: r.Emoji, Count: r.Count, ReactedByMe: r.ReactedByMe}
		}
		out[id] = conv
	}
	return out, nil
}
