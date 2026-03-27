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

func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMembers(ctx, groupID, role, limit, cursor)
	}
	return r.fallback.ListMembers(ctx, groupID, role, limit, cursor)
}

func (r *PostgresRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.SearchMembers(ctx, groupID, keyword, role, limit, cursor)
	}
	return r.fallback.SearchMembers(ctx, groupID, keyword, role, limit, cursor)
}

func (r *PostgresRepository) ListMyGroups(ctx context.Context, userID string, limit int, cursor string) ([]GroupItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMyGroups(ctx, userID, limit, cursor)
	}
	return r.fallback.ListMyGroups(ctx, userID, limit, cursor)
}

func (r *PostgresRepository) JoinGroup(ctx context.Context, userID string, groupID string, password *string, message *string) (JoinGroupResult, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.JoinGroup(ctx, userID, groupID, password, message)
	}
	return r.fallback.JoinGroup(ctx, userID, groupID, password, message)
}

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return r.fallback.CreateAnnouncement(ctx, creatorID, req)
}

func (r *PostgresRepository) ListAnnouncements(ctx context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	return r.fallback.ListAnnouncements(ctx, groupID)
}

func (r *PostgresRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	return r.fallback.UpdateAnnouncement(ctx, groupID, req)
}

func (r *PostgresRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	return r.fallback.DeleteAnnouncement(ctx, groupID, announcementID)
}

func (r *PostgresRepository) ListApplications(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	return r.fallback.ListApplications(ctx, groupID, status, limit, cursor)
}

func (r *PostgresRepository) ApproveApplication(ctx context.Context, approverID string, applicationID string) error {
	return r.fallback.ApproveApplication(ctx, approverID, applicationID)
}

func (r *PostgresRepository) RejectApplication(ctx context.Context, applicationID string, reason *string) error {
	return r.fallback.RejectApplication(ctx, applicationID, reason)
}

func (r *PostgresRepository) UpdateMySettings(ctx context.Context, userID string, req UpdateGroupMySettingsRequest) error {
	return r.fallback.UpdateMySettings(ctx, userID, req)
}

func (r *PostgresRepository) LeaveGroup(ctx context.Context, userID string, groupID string) error {
	return r.fallback.LeaveGroup(ctx, userID, groupID)
}

func (r *PostgresRepository) ListMyApplications(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	return r.fallback.ListMyApplications(ctx, userID, status, limit, cursor)
}

func (r *PostgresRepository) CancelApplication(ctx context.Context, applicationID string) error {
	return r.fallback.CancelApplication(ctx, applicationID)
}

func (r *PostgresRepository) ListMyInvites(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	return r.fallback.ListMyInvites(ctx, userID, status, limit, cursor)
}

func (r *PostgresRepository) AcceptInvite(ctx context.Context, userID string, inviteID string) error {
	return r.fallback.AcceptInvite(ctx, userID, inviteID)
}

func (r *PostgresRepository) RejectInvite(ctx context.Context, inviteID string) error {
	return r.fallback.RejectInvite(ctx, inviteID)
}

func (r *PostgresRepository) InviteMember(ctx context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	return r.fallback.InviteMember(ctx, userID, req)
}

func (r *PostgresRepository) ListSentInvites(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	return r.fallback.ListSentInvites(ctx, groupID, status, limit, cursor)
}

func (r *PostgresRepository) RevokeInvite(ctx context.Context, groupID string, inviteID string) error {
	return r.fallback.RevokeInvite(ctx, groupID, inviteID)
}

func (r *PostgresRepository) SetMemberRole(ctx context.Context, userID string, req SetMemberRoleRequest) error {
	return r.fallback.SetMemberRole(ctx, userID, req)
}

func (r *PostgresRepository) MuteMember(ctx context.Context, userID string, req MuteMemberRequest) error {
	return r.fallback.MuteMember(ctx, userID, req)
}

func (r *PostgresRepository) UnmuteMember(ctx context.Context, userID string, req UnmuteMemberRequest) error {
	return r.fallback.UnmuteMember(ctx, userID, req)
}

func (r *PostgresRepository) BanMember(ctx context.Context, userID string, req BanMemberRequest) error {
	return r.fallback.BanMember(ctx, userID, req)
}

func (r *PostgresRepository) UnbanMember(ctx context.Context, userID string, req UnbanMemberRequest) error {
	return r.fallback.UnbanMember(ctx, userID, req)
}

func (r *PostgresRepository) ListBannedMembers(ctx context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	return r.fallback.ListBannedMembers(ctx, groupID, limit, cursor)
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, userID string, req RemoveMemberRequest) error {
	return r.fallback.RemoveMember(ctx, userID, req)
}

func (r *PostgresRepository) TransferOwnership(ctx context.Context, userID string, req TransferOwnershipRequest) error {
	return r.fallback.TransferOwnership(ctx, userID, req)
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
