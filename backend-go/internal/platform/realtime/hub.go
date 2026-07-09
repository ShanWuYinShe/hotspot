package realtime

import (
	"context"
	"sync"
)

type FanoutSink interface {
	Send(Event) error
	Close() error
}

// maxConnectionsPerBroadcast caps the pre-allocation in deliverAllLocal to
// prevent a sudden allocation spike when broadcasting to a hub with a very
// large number of connections. BroadcastToAll is rare (system events only),
// so the O(n) iteration is acceptable.
const maxConnectionsPerBroadcast = 1000

type Hub struct {
	mu             sync.RWMutex
	byUser         map[string][]FanoutSink
	bySession      map[string][]FanoutSink
	byUserWatchers map[string][]FanoutSink
	byGroup        map[string][]FanoutSink
	publisher      EventPublisher
}

func NewHub() *Hub {
	return &Hub{byUser: map[string][]FanoutSink{}, bySession: map[string][]FanoutSink{}, byUserWatchers: map[string][]FanoutSink{}, byGroup: map[string][]FanoutSink{}}
}

func (h *Hub) SetPublisher(p EventPublisher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.publisher = p
}

func (h *Hub) RegisterUser(userID, sessionID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	first := len(h.byUser[userID]) == 0
	h.byUser[userID] = append(h.byUser[userID], sink)
	h.bySession[sessionID] = append(h.bySession[sessionID], sink)
	return first
}

func (h *Hub) UnregisterUser(userID, sessionID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	remove := func(m map[string][]FanoutSink, key string) bool {
		sinks := m[key]
		out := sinks[:0]
		for _, s := range sinks {
			if s != sink {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			delete(m, key)
			return true
		}
		m[key] = out
		return false
	}
	remove(h.bySession, sessionID)
	return remove(h.byUser, userID)
}

func (h *Hub) CloseSession(sessionID string, event Event) {
	h.closeSessionLocal(sessionID, event)
	h.publish("session:"+sessionID, event)
}

func (h *Hub) closeSessionLocal(sessionID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.bySession[sessionID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
		_ = sink.Close()
	}
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
	h.deliverUserLocal(userID, event)
	h.publish("user:"+userID, event)
}

func (h *Hub) deliverUserLocal(userID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byUser[userID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

func (h *Hub) BroadcastToUserSubscribers(userID string, event Event) {
	h.deliverUserSubscribersLocal(userID, event)
	h.publish("usersub:"+userID, event)
}

func (h *Hub) deliverUserSubscribersLocal(userID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byUserWatchers[userID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

func (h *Hub) BroadcastToGroup(groupID string, event Event) {
	h.deliverGroupLocal(groupID, event)
	h.publish("group:"+groupID, event)
}

func (h *Hub) deliverGroupLocal(groupID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byGroup[groupID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

// BroadcastToAll sends event to every connected session. It is O(n) for all
// connections — it collects a snapshot of all sinks under RLock then iterates
// outside the lock. This is used for rare system-wide events only; it should
// not be called on hot paths.
func (h *Hub) BroadcastToAll(event Event) {
	h.deliverAllLocal(event)
	h.publish("all", event)
}

// deliverAllLocal collects a snapshot of every connected sink under RLock and
// then delivers to each outside the lock. The pre-allocation is capped at
// maxConnectionsPerBroadcast to avoid unbounded allocation spikes. This is
// O(n) for all active connections.
func (h *Hub) deliverAllLocal(event Event) {
	h.mu.RLock()
	n := len(h.bySession)
	if n > maxConnectionsPerBroadcast {
		n = maxConnectionsPerBroadcast
	}
	sinks := make([]FanoutSink, 0, n)
	for _, sessionSinks := range h.bySession {
		sinks = append(sinks, sessionSinks...)
	}
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

func (h *Hub) publish(channel string, event Event) {
	h.mu.RLock()
	p := h.publisher
	h.mu.RUnlock()
	if p != nil {
		_ = p.Publish(context.Background(), channel, event)
	}
}
