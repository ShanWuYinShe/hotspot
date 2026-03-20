package realtime

import "sync"

type FanoutSink interface{ Send(Event) error }

type Hub struct {
	mu             sync.RWMutex
	byUser         map[string][]FanoutSink
	byUserWatchers map[string][]FanoutSink
	byGroup        map[string][]FanoutSink
}

func NewHub() *Hub {
	return &Hub{byUser: map[string][]FanoutSink{}, byUserWatchers: map[string][]FanoutSink{}, byGroup: map[string][]FanoutSink{}}
}

func (h *Hub) RegisterUser(userID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	first := len(h.byUser[userID]) == 0
	h.byUser[userID] = append(h.byUser[userID], sink)
	return first
}
func (h *Hub) UnregisterUser(userID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	sinks := h.byUser[userID]
	out := sinks[:0]
	for _, s := range sinks {
		if s != sink {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		delete(h.byUser, userID)
		return true
	}
	h.byUser[userID] = out
	return false
}
func (h *Hub) SetUserSubscriptions(userIDs []string, sink FanoutSink) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for uid, sinks := range h.byUserWatchers {
		out := sinks[:0]
		for _, s := range sinks {
			if s != sink {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			delete(h.byUserWatchers, uid)
		} else {
			h.byUserWatchers[uid] = out
		}
	}
	for _, uid := range userIDs {
		h.byUserWatchers[uid] = append(h.byUserWatchers[uid], sink)
	}
}
func (h *Hub) SetGroupSubscriptions(groupIDs []string, sink FanoutSink) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for gid, sinks := range h.byGroup {
		out := sinks[:0]
		for _, s := range sinks {
			if s != sink {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			delete(h.byGroup, gid)
		} else {
			h.byGroup[gid] = out
		}
	}
	for _, gid := range groupIDs {
		h.byGroup[gid] = append(h.byGroup[gid], sink)
	}
}
func (h *Hub) BroadcastToUser(userID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byUser[userID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}
func (h *Hub) BroadcastToUserSubscribers(userID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byUserWatchers[userID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}
func (h *Hub) BroadcastToGroup(groupID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byGroup[groupID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}
