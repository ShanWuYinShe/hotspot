package realtime

import "sync"

type subscriptions struct {
	mu       sync.RWMutex
	userIDs  map[string]bool
	groupIDs map[string]bool
}

func newSubscriptions() *subscriptions {
	return &subscriptions{userIDs: map[string]bool{}, groupIDs: map[string]bool{}}
}

func (s *subscriptions) snapshot() SubscriptionsSnapshotPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	payload := SubscriptionsSnapshotPayload{}
	for id := range s.userIDs {
		payload.UserIDs = append(payload.UserIDs, id)
	}
	for id := range s.groupIDs {
		payload.GroupIDs = append(payload.GroupIDs, id)
	}
	return payload
}

func (s *subscriptions) apply(op, kind string, ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.userIDs
	if kind == "group" {
		target = s.groupIDs
	}
	switch op {
	case "set":
		for k := range target {
			delete(target, k)
		}
		for _, id := range ids {
			target[id] = true
		}
	case "subscribe":
		for _, id := range ids {
			target[id] = true
		}
	case "unsubscribe":
		for _, id := range ids {
			delete(target, id)
		}
	}
}
