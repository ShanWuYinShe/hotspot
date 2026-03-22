package users

import (
	"context"
	"sort"
	"sync"
)

type InMemoryRepository struct {
	mu        sync.RWMutex
	profiles  map[string]MyProfile
	settings  map[string]UserSettings
	following map[string]map[string]bool
	blocked   map[string]map[string]bool
	remarks   map[string]map[string]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{profiles: make(map[string]MyProfile), settings: make(map[string]UserSettings), following: make(map[string]map[string]bool), blocked: make(map[string]map[string]bool), remarks: make(map[string]map[string]string)}
}

func (r *InMemoryRepository) GetProfile(_ context.Context, userID string) (MyProfile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, ok := r.profiles[userID]
	return profile, ok, nil
}
func (r *InMemoryRepository) PutProfile(_ context.Context, profile MyProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.UserID] = profile
	return nil
}
func (r *InMemoryRepository) DeleteProfile(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.profiles, userID)
	delete(r.settings, userID)
	delete(r.following, userID)
	delete(r.blocked, userID)
	delete(r.remarks, userID)
	return nil
}
func (r *InMemoryRepository) GetSettings(_ context.Context, userID string) (UserSettings, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	settings, ok := r.settings[userID]
	return settings, ok, nil
}
func (r *InMemoryRepository) PutSettings(_ context.Context, userID string, settings UserSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings[userID] = settings
	return nil
}
func (r *InMemoryRepository) SearchProfiles(_ context.Context, req SearchUsersRequest) ([]MyProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []MyProfile{}
	for _, profile := range r.profiles {
		if req.Keyword == "" || profile.Nickname == req.Keyword {
			items = append(items, profile)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UserID < items[j].UserID })
	return items, nil
}
func (r *InMemoryRepository) Follow(_ context.Context, userID, targetUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.following[userID] == nil {
		r.following[userID] = map[string]bool{}
	}
	r.following[userID][targetUserID] = true
	return nil
}
func (r *InMemoryRepository) Unfollow(_ context.Context, userID, targetUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.following[userID] != nil {
		delete(r.following[userID], targetUserID)
	}
	return nil
}
func (r *InMemoryRepository) Block(_ context.Context, userID, targetUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.blocked[userID] == nil {
		r.blocked[userID] = map[string]bool{}
	}
	r.blocked[userID][targetUserID] = true
	return nil
}
func (r *InMemoryRepository) Unblock(_ context.Context, userID, targetUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.blocked[userID] != nil {
		delete(r.blocked[userID], targetUserID)
	}
	return nil
}
func (r *InMemoryRepository) SetRemark(_ context.Context, userID, targetUserID, remark string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.remarks[userID] == nil {
		r.remarks[userID] = map[string]string{}
	}
	r.remarks[userID][targetUserID] = remark
	return nil
}
func (r *InMemoryRepository) ListBlocked(_ context.Context, userID string) ([]BlockedUserItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []BlockedUserItem{}
	for targetUserID := range r.blocked[userID] {
		items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: targetUserID}, Remark: r.remarks[userID][targetUserID]})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].User.UserID < items[j].User.UserID })
	return items, nil
}
func (r *InMemoryRepository) Relationship(_ context.Context, userID, targetUserID string) (Relationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.blocked[userID] != nil && r.blocked[userID][targetUserID] || r.blocked[targetUserID] != nil && r.blocked[targetUserID][userID] {
		return RelationshipBlocked, nil
	}
	if r.following[userID] != nil && r.following[userID][targetUserID] && r.following[targetUserID] != nil && r.following[targetUserID][userID] {
		return RelationshipFriend, nil
	}
	if r.following[userID] != nil && r.following[userID][targetUserID] {
		return RelationshipFollowing, nil
	}
	if r.following[targetUserID] != nil && r.following[targetUserID][userID] {
		return RelationshipFollower, nil
	}
	return RelationshipNone, nil
}
