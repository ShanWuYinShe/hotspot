package groups

import (
	"context"
	"errors"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Get(ctx, groupID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupItem{}, false, persistence.ErrNotImplemented
	}
	item, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupItem{}, false, nil
		}
		return GroupItem{}, false, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	myState, err := loadGroupMembership(ctx, queryable, groupID, principal.UserID)
	if err != nil {
		return GroupItem{}, false, err
	}
	item.MyState = myState
	item.Group.MembersCount, err = countGroupMembers(ctx, queryable, groupID)
	if err != nil {
		return GroupItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) Search(ctx context.Context, keyword string) (PaginatedGroups, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Search(ctx, keyword)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, querySearchGroups, keyword)
	if err != nil {
		return PaginatedGroups{}, err
	}
	defer rows.Close()
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items := []GroupItem{}
	for rows.Next() {
		item, err := scanGroupBase(rows)
		if err != nil {
			return PaginatedGroups{}, err
		}
		item.MyState, err = loadGroupMembership(ctx, queryable, item.Group.GroupID, principal.UserID)
		if err != nil {
			return PaginatedGroups{}, err
		}
		item.Group.MembersCount, err = countGroupMembers(ctx, queryable, item.Group.GroupID)
		if err != nil {
			return PaginatedGroups{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroups{}, err
	}
	return PaginatedGroups{Items: items}, nil
}

func scanGroupBase(scanner interface{ Scan(...any) error }) (GroupItem, error) {
	var item GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers); err != nil {
		return GroupItem{}, err
	}
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.JoinMethod = GroupJoinMethod(joinMethod)
	return item, nil
}

func getGroupBase(ctx context.Context, queryable persistence.QueryableDB, groupID string) (GroupItem, error) {
	return scanGroupBase(queryable.QueryRow(ctx, queryGetGroup, groupID))
}

func loadGroupMembership(ctx context.Context, queryable persistence.QueryableDB, groupID, userID string) (*MyGroupState, error) {
	if userID == "" {
		return nil, nil
	}
	var role string
	var joinedAt time.Time
	if err := queryable.QueryRow(ctx, queryGetGroupMembership, groupID, userID).Scan(&role, &joinedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &MyGroupState{Role: GroupRole(role), IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: joinedAt.UTC().Format(time.RFC3339)}, nil
}

func countGroupMembers(ctx context.Context, queryable persistence.QueryableDB, groupID string) (int, error) {
	var count int
	if err := queryable.QueryRow(ctx, queryCountGroupMembers, groupID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
