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
	// AreMembers 批量判定 userID 是否为各群成员（单次往返）。WS 订阅鉴权
	// 消费：订阅消息最坏携带 100 个群 ID，逐条 IsMember 会把读循环阻塞成
	// 串行 DB 往返。缺失的 group_id 不在返回 map 中（非成员）。
	AreMembers(context.Context, []string, string) (map[string]bool, error)
	IsBanned(context.Context, string, string) (bool, error)
	GetApplication(context.Context, string) (GroupApplicationItem, bool, error)
	ListMyGroups(context.Context, string, int, string) ([]GroupItem, string, error)
	// JoinGroup password 参数已移除：密码比对是业务规则，收口在 Service
	// （经 GetGroupPasswordHash 取哈希后 bcrypt 比对），repo 只负责入群写入。
	JoinGroup(context.Context, string, string, *string) (EnterGroupResult, error)
	// GetGroupPasswordHash 返回群密码 bcrypt 哈希（无密码群返回 nil, nil）。
	GetGroupPasswordHash(context.Context, string) (*string, error)
	CreateAnnouncement(context.Context, string, CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error)
	ListAnnouncements(context.Context, string, int) ([]GroupAnnouncementItem, error) // limit<=0 取默认 10，>50 截断硬上限
	// GetAnnouncement 按归属双键（group_id + announcement_id）直查单条公告。
	// UpdateAnnouncement 更新后取回完整对象用它，替代全量 List 再线性扫描
	// 找回单条（审查 P2-2）。不存在/跨群归属时 found=false。
	GetAnnouncement(ctx context.Context, groupID, announcementID string) (item GroupAnnouncementItem, found bool, err error)
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
	// CountOwnedGroups 统计用户担任群主的未解散群数（注销守卫消费）。
	CountOwnedGroups(context.Context, string) (int, error)
}
