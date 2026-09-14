package groups

import (
	"context"
	"time"
)

type Repository interface {
	Create(context.Context, string, CreateGroupRequest) (GroupItem, error)
	Get(context.Context, string) (GroupItem, bool, error)
	Search(context.Context, SearchGroupsRequest) (PaginatedGroups, error)
	Update(context.Context, string, UpdateGroupRequest) error
	Dismiss(context.Context, string) error
	ListMembers(context.Context, string, *GroupRole, int, string) ([]GroupMemberItem, string, error)
	ListAdminMembers(context.Context, string, *GroupRole, int, string) ([]AdminGroupMemberItem, string, error)
	SearchMembers(context.Context, string, string, *GroupRole, int, string) ([]GroupMemberItem, string, error)
	IsMember(context.Context, string, string) (bool, error)
	IsBanned(context.Context, string, string) (bool, error)
	GetApplication(context.Context, string) (GroupApplicationItem, bool, error)
	ListMyGroups(context.Context, string, int, string) ([]GroupItem, string, error)
	JoinGroup(context.Context, string, string, *string, *string) (EnterGroupResult, error)
	CreateAnnouncement(context.Context, string, CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error)
	ListAnnouncements(context.Context, string) ([]GroupAnnouncementItem, error)
	UpdateAnnouncement(context.Context, string, UpdateGroupAnnouncementRequest) error
	DeleteAnnouncement(context.Context, string, string) error
	ListApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error)
	ApproveApplication(context.Context, string, string) error
	RejectApplication(context.Context, string, *string) error
	UpdatePreferences(context.Context, string, UpdateGroupPreferencesRequest) error
	LeaveGroup(context.Context, string, string) error
	ListMyApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error)
	CancelApplication(context.Context, string) error
	// GetInvite 按 invite_id 直查邀请（对照 GetApplication：替代按列表
	// limit=1000 的截断扫描）。invitee 为被邀请人 user_id，供归属校验与撤销
	// 通知寻址。邀请不存在时 found=false。
	GetInvite(ctx context.Context, inviteID string) (invite GroupInviteItem, invitee string, found bool, err error)
	ListMyInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupInviteItem, string, error)
	AcceptInvite(context.Context, string, string) error
	RejectInvite(context.Context, string) error
	InviteMember(context.Context, string, InviteMemberRequest, *time.Time) (Invitation, error)
	ListSentInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupSentInviteItem, string, error)
	RevokeInvite(context.Context, string, string) error
	SetMemberRole(context.Context, string, SetMemberRoleRequest) error
	MuteMember(context.Context, string, MuteMemberRequest) error
	UnmuteMember(context.Context, string, UnmuteMemberRequest) error
	BanMember(context.Context, string, BanMemberRequest) error
	UnbanMember(context.Context, string, UnbanMemberRequest) error
	ListBannedMembers(context.Context, string, int, string) ([]AdminBannedMemberItem, string, error)
	RemoveMember(context.Context, string, RemoveMemberRequest) error
	TransferOwnership(context.Context, string, TransferOwnershipRequest) error
	// ListGroupsByUser returns a user's visible groups for the profile
	// aggregation consumed by the users module via Service (see
	// Service.ListGroupsByUser).
	ListGroupsByUser(context.Context, string, int, string) (PaginatedGroups, error)
	CountCommonGroups(context.Context, string, string) (int, error)
}
