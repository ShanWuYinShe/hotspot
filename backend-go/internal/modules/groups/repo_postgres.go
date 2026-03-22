package groups

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Create(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	groupID := fmt.Sprintf("group_%d", now.UnixNano())
	isSearchable := true
	if req.IsSearchable != nil {
		isSearchable = *req.IsSearchable
	}
	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = 200
	}
	joinMethod := req.JoinMethod
	if joinMethod == "" {
		joinMethod = "open"
	}
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1}, Description: req.Description, Tags: req.Tags, HasPassword: req.Password != "", Location: req.Location, MyState: &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: now.Format(time.RFC3339)}, JoinMethod: joinMethod, IsSearchable: isSearchable, MaxMembers: maxMembers}
	if err := queryable.Exec(ctx, queryInsertGroup, groupID, req.Name, req.AvatarFileID, req.Description, req.Password != "", req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, joinMethod, isSearchable, maxMembers); err != nil {
		return GroupItem{}, err
	}
	_ = userID
	return item, nil
}
