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

func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdateGroupRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Update(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var hasPassword *bool
	if req.ClearPassword != nil && *req.ClearPassword {
		cleared := false
		hasPassword = &cleared
	} else if req.Password != nil {
		hasPwd := *req.Password != ""
		hasPassword = &hasPwd
	}
	var locationName *string
	var latitude *float64
	var longitude *float64
	if req.Location != nil {
		locationName = &req.Location.LocationName
		latitude = &req.Location.Point.Latitude
		longitude = &req.Location.Point.Longitude
	}
	_ = userID
	err := queryable.Exec(ctx, queryUpdateGroup, req.GroupID, nilOrString(req.Name), nilOrString(req.AvatarFileID), nilOrString(req.Description), hasPassword, nilOrStringJoinMethod(req.JoinMethod), nilOrBool(req.IsSearchable), nilOrInt(req.MaxMembers), locationName, latitude, longitude)
	return err
}

func (r *PostgresRepository) Dismiss(ctx context.Context, groupID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Dismiss(ctx, groupID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	err := queryable.Exec(ctx, queryDeleteGroup, groupID)
	return err
}

func nilOrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilOrStringJoinMethod(g *GroupJoinMethod) *string {
	if g == nil {
		return nil
	}
	s := string(*g)
	return &s
}

func nilOrBool(b *bool) *bool {
	return b
}

func nilOrInt(i *int) *int {
	return i
}
