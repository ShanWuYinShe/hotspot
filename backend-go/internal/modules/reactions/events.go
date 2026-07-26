package reactions

// Reaction realtime event types (spec api_spec_websocket.md). Defined here so
// the reactions package has no dependency on the realtime package's constant
// block growing unbounded; these mirror realtime.EventTypeReaction* values
// but live with the domain that emits them.
const (
	EventTypeReactionAdded   = "ReactionAdded"
	EventTypeReactionRemoved = "ReactionRemoved"
)
