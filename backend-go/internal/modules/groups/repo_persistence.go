package groups

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	return r.impl.Get(ctx, groupID)
}

func (r *PersistenceRepository) Search(ctx context.Context, req SearchGroupsRequest) (PaginatedGroups, error) {
	return r.impl.Search(ctx, req)
}

func (r *PersistenceRepository) Update(ctx context.Context, userID string, req UpdateGroupRequest) error {
	return r.impl.Update(ctx, userID, req)
}

func (r *PersistenceRepository) Dismiss(ctx context.Context, groupID string) error {
	return r.impl.Dismiss(ctx, groupID)
}

func (r *PersistenceRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	return r.impl.ListMembers(ctx, groupID, role, limit, cursor)
}

func (r *PersistenceRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	return r.impl.SearchMembers(ctx, groupID, keyword, role, limit, cursor)
}

func (r *PersistenceRepository) ListMyGroups(ctx context.Context, userID string, limit int, cursor string) ([]GroupItem, string, error) {
	return r.impl.ListMyGroups(ctx, userID, limit, cursor)
}

func (r *PersistenceRepository) JoinGroup(ctx context.Context, userID string, groupID string, password *string, message *string) (JoinGroupResult, error) {
	return r.impl.JoinGroup(ctx, userID, groupID, password, message)
}

func (r *PersistenceRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return r.impl.CreateAnnouncement(ctx, creatorID, req)
}

func (r *PersistenceRepository) ListAnnouncements(ctx context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	return r.impl.ListAnnouncements(ctx, groupID)
}

func (r *PersistenceRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	return r.impl.UpdateAnnouncement(ctx, groupID, req)
}

func (r *PersistenceRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	return r.impl.DeleteAnnouncement(ctx, groupID, announcementID)
}

func (r *PersistenceRepository) ListApplications(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	return r.impl.ListApplications(ctx, groupID, status, limit, cursor)
}

func (r *PersistenceRepository) ApproveApplication(ctx context.Context, approverID string, applicationID string) error {
	return r.impl.ApproveApplication(ctx, approverID, applicationID)
}

func (r *PersistenceRepository) RejectApplication(ctx context.Context, applicationID string, reason *string) error {
	return r.impl.RejectApplication(ctx, applicationID, reason)
}

func (r *PersistenceRepository) UpdateMySettings(ctx context.Context, userID string, req UpdateGroupMySettingsRequest) error {
	return r.impl.UpdateMySettings(ctx, userID, req)
}

func (r *PersistenceRepository) LeaveGroup(ctx context.Context, userID string, groupID string) error {
	return r.impl.LeaveGroup(ctx, userID, groupID)
}

func (r *PersistenceRepository) ListMyApplications(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	return r.impl.ListMyApplications(ctx, userID, status, limit, cursor)
}

func (r *PersistenceRepository) CancelApplication(ctx context.Context, applicationID string) error {
	return r.impl.CancelApplication(ctx, applicationID)
}

func (r *PersistenceRepository) ListMyInvites(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	return r.impl.ListMyInvites(ctx, userID, status, limit, cursor)
}

func (r *PersistenceRepository) AcceptInvite(ctx context.Context, userID string, inviteID string) error {
	return r.impl.AcceptInvite(ctx, userID, inviteID)
}

func (r *PersistenceRepository) RejectInvite(ctx context.Context, inviteID string) error {
	return r.impl.RejectInvite(ctx, inviteID)
}

func (r *PersistenceRepository) InviteMember(ctx context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	return r.impl.InviteMember(ctx, userID, req)
}

func (r *PersistenceRepository) ListSentInvites(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	return r.impl.ListSentInvites(ctx, groupID, status, limit, cursor)
}

func (r *PersistenceRepository) RevokeInvite(ctx context.Context, groupID string, inviteID string) error {
	return r.impl.RevokeInvite(ctx, groupID, inviteID)
}

func (r *PersistenceRepository) SetMemberRole(ctx context.Context, userID string, req SetMemberRoleRequest) error {
	return r.impl.SetMemberRole(ctx, userID, req)
}

func (r *PersistenceRepository) MuteMember(ctx context.Context, userID string, req MuteMemberRequest) error {
	return r.impl.MuteMember(ctx, userID, req)
}

func (r *PersistenceRepository) UnmuteMember(ctx context.Context, userID string, req UnmuteMemberRequest) error {
	return r.impl.UnmuteMember(ctx, userID, req)
}

func (r *PersistenceRepository) BanMember(ctx context.Context, userID string, req BanMemberRequest) error {
	return r.impl.BanMember(ctx, userID, req)
}

func (r *PersistenceRepository) UnbanMember(ctx context.Context, userID string, req UnbanMemberRequest) error {
	return r.impl.UnbanMember(ctx, userID, req)
}

func (r *PersistenceRepository) ListBannedMembers(ctx context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	return r.impl.ListBannedMembers(ctx, groupID, limit, cursor)
}

func (r *PersistenceRepository) RemoveMember(ctx context.Context, userID string, req RemoveMemberRequest) error {
	return r.impl.RemoveMember(ctx, userID, req)
}

func (r *PersistenceRepository) TransferOwnership(ctx context.Context, userID string, req TransferOwnershipRequest) error {
	return r.impl.TransferOwnership(ctx, userID, req)
}

func (r *PersistenceRepository) IsMember(ctx context.Context, groupID string, userID string) (bool, error) {
	return r.impl.IsMember(ctx, groupID, userID)
}
