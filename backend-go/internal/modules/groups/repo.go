package groups

import "context"

type Repository interface {
	Create(context.Context, string, CreateGroupRequest) (GroupItem, error)
	Get(context.Context, string) (GroupItem, bool, error)
	Search(context.Context, SearchGroupsRequest) (PaginatedGroups, error)
	Update(context.Context, string, UpdateGroupRequest) error
	Dismiss(context.Context, string) error
	ListMembers(context.Context, string, *GroupRole, int, string) ([]GroupMemberItem, string, error)
	SearchMembers(context.Context, string, string, *GroupRole, int, string) ([]GroupMemberItem, string, error)
	ListMyGroups(context.Context, string, int, string) ([]GroupItem, string, error)
	JoinGroup(context.Context, string, string, *string, *string) (JoinGroupResult, error)
	CreateAnnouncement(context.Context, string, CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error)
	ListAnnouncements(context.Context, string) ([]GroupAnnouncementItem, error)
	UpdateAnnouncement(context.Context, string, UpdateGroupAnnouncementRequest) error
	DeleteAnnouncement(context.Context, string, string) error
	ListApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error)
	ApproveApplication(context.Context, string, string) error
	RejectApplication(context.Context, string, *string) error
	UpdateMySettings(context.Context, string, UpdateGroupMySettingsRequest) error
	LeaveGroup(context.Context, string, string) error
	ListMyApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error)
	CancelApplication(context.Context, string) error
	ListMyInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupInviteItem, string, error)
	AcceptInvite(context.Context, string, string) error
	RejectInvite(context.Context, string) error
	InviteMember(context.Context, string, InviteMemberRequest) (Invitation, error)
	ListSentInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupSentInviteItem, string, error)
	RevokeInvite(context.Context, string, string) error
	SetMemberRole(context.Context, string, SetMemberRoleRequest) error
	MuteMember(context.Context, string, MuteMemberRequest) error
	UnmuteMember(context.Context, string, UnmuteMemberRequest) error
	BanMember(context.Context, string, BanMemberRequest) error
	UnbanMember(context.Context, string, UnbanMemberRequest) error
	ListBannedMembers(context.Context, string, int, string) ([]BannedMemberItem, string, error)
	RemoveMember(context.Context, string, RemoveMemberRequest) error
	TransferOwnership(context.Context, string, TransferOwnershipRequest) error
	IsMember(context.Context, string, string) (bool, error)
}
