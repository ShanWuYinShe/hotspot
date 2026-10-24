// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey, pick, type RequestOptions } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	GroupAnnouncementItem,
	GroupItem,
	GroupMemberItem,
	AdminGroupMemberItem,
	GroupSettings,
	JoinGroupResult,
	GroupApplicationItem,
	GroupInviteItem,
	GroupSentInviteItem,
	AdminBannedMemberItem,
	Invitation,
	InviteAcceptedResult,
	SearchGroupsRequest,
	UpdateGroupRequest,
	DismissGroupRequest,
	TransferOwnershipRequest,
	SearchGroupMembersRequest,
	CreateGroupInviteRequest,
	ListSentInvitesRequest,
	RevokeGroupInviteRequest,
	SetMemberRoleRequest,
	SetMemberMuteRequest,
	SetMemberBanRequest,
	BannedMembersRequest,
	RemoveMemberRequest,
	GroupApplicationsRequest,
	ApproveGroupApplicationRequest,
	RejectGroupApplicationRequest,
	CreateGroupAnnouncementRequest,
	UpdateGroupAnnouncementRequest,
	DeleteGroupAnnouncementRequest,
	ListSubmittedApplicationsRequest,
	CancelApplicationRequest,
	ListReceivedInvitesRequest,
	AcceptInviteRequest,
	RejectInviteRequest,
	PaginatedData,
	CreateGroupRequest,
	UpdateGroupPreferencesRequest,
	ListAdminGroupMembersRequest,
	RecallGroupMessageAdminRequest,
	PaginationParams,
} from '@/api/types';

// Groups
export async function getGroupDetail(groupId: string)     { return request<GroupItem>(pick(Endpoints.Groups.GetDetail, Endpoints.Public.Groups.GetDetail), { group_id: groupId }); }
export async function createGroup(params: CreateGroupRequest) { return request<GroupItem>(Endpoints.Groups.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// spec: api_spec_group.md → groups/fetch_joined（成员视角，viewer_state 必非 null）
export async function listMyGroups(params?: { pagination?: PaginationParams })       { return request<PaginatedData<GroupItem>>(Endpoints.Groups.ListJoined, params ?? {}); }
// spec: api_spec_group.md → groups/enter（direct 直接加入 / approval 创建申请）、groups/exit
export async function joinGroup(groupId: string, extra?: { password?: string; message?: string }) {
	return request<JoinGroupResult>(Endpoints.Groups.Join, { group_id: groupId, ...extra });
}
export async function leaveGroup(groupId: string)          { return request<void>(Endpoints.Groups.Leave, { group_id: groupId }); }

export async function updateGroupPreferences(params: UpdateGroupPreferencesRequest) { return request<void>(Endpoints.Groups.UpdatePreferences, params); }

// Groups extended
export async function getGroupMembers(groupId: string, pagination?: PaginationParams) {
  return request<PaginatedData<GroupMemberItem>>(pick(Endpoints.Groups.Members.List, Endpoints.Public.Groups.Members.List), { group_id: groupId, pagination });
}
export async function getGroupAnnouncements(groupId: string) {
  return request<{ items: GroupAnnouncementItem[] }>(Endpoints.Groups.Announcements.List, { group_id: groupId });
}

// Groups admin（spec: api_spec_group.md → 群组管理：搜索/更新/解散/转让/申请/邀请/成员/公告）
export async function searchGroups(params: SearchGroupsRequest, opts?: RequestOptions) { return request<PaginatedData<GroupItem>>(pick(Endpoints.Groups.Search, Endpoints.Public.Groups.Search), params, {}, 0, undefined, opts); }
export async function updateGroup(params: UpdateGroupRequest) { return request<GroupItem>(Endpoints.Groups.Admin.Update, params); }
export async function dismissGroup(params: DismissGroupRequest) { return request<void>(Endpoints.Groups.Admin.Dismiss, params); }
export async function transferGroupOwnership(params: TransferOwnershipRequest) { return request<void>(Endpoints.Groups.Admin.TransferOwnership, params); }
export async function listMyApplications(params?: ListSubmittedApplicationsRequest) { return request<PaginatedData<GroupApplicationItem>>(Endpoints.Groups.ListSubmittedApplications, params ?? {}); }
export async function cancelGroupApplication(params: CancelApplicationRequest) { return request<void>(Endpoints.Groups.CancelApplication, params); }
export async function listMyInvites(params?: ListReceivedInvitesRequest) { return request<PaginatedData<GroupInviteItem>>(Endpoints.Groups.ListReceivedInvites, params ?? {}); }
export async function acceptGroupInvite(params: AcceptInviteRequest) { return request<InviteAcceptedResult>(Endpoints.Groups.AcceptInvite, params); }
export async function rejectGroupInvite(params: RejectInviteRequest) { return request<void>(Endpoints.Groups.RejectInvite, params); }

export async function searchGroupMembers(params: SearchGroupMembersRequest) { return request<PaginatedData<GroupMemberItem>>(pick(Endpoints.Groups.Members.Search, Endpoints.Public.Groups.Members.Search), params); }
export async function inviteGroupMember(params: CreateGroupInviteRequest) { return request<Invitation>(Endpoints.Groups.Admin.Invites.Create, params); }
export async function setGroupMemberRole(params: SetMemberRoleRequest) { return request<void>(Endpoints.Groups.Admin.Members.SetRole, params); }
export async function setGroupMemberMute(params: SetMemberMuteRequest) { return request<void>(Endpoints.Groups.Admin.Members.SetMute, params); }
export async function banGroupMember(params: Omit<SetMemberBanRequest, 'active'>) { return request<void>(Endpoints.Groups.Admin.Members.SetBan, { ...params, active: true }); }
export async function unbanGroupMember(params: Omit<SetMemberBanRequest, 'active'>) { return request<void>(Endpoints.Groups.Admin.Members.SetBan, { ...params, active: false }); }
export async function listBannedGroupMembers(params: BannedMembersRequest) { return request<PaginatedData<AdminBannedMemberItem>>(Endpoints.Groups.Admin.Members.ListBanned, params); }
export async function removeGroupMember(params: RemoveMemberRequest) { return request<void>(Endpoints.Groups.Admin.Members.Remove, params); }
export async function listSentGroupInvites(params: ListSentInvitesRequest) { return request<PaginatedData<GroupSentInviteItem>>(Endpoints.Groups.Admin.Invites.ListSent, params); }
export async function revokeSentGroupInvite(params: RevokeGroupInviteRequest) { return request<void>(Endpoints.Groups.Admin.Invites.Revoke, params); }

export async function listGroupApplications(params: GroupApplicationsRequest) { return request<PaginatedData<GroupApplicationItem>>(Endpoints.Groups.Admin.Applications.List, params); }
export async function approveGroupApplication(params: ApproveGroupApplicationRequest) { return request<void>(Endpoints.Groups.Admin.Applications.Approve, params); }
export async function rejectGroupApplication(params: RejectGroupApplicationRequest) { return request<void>(Endpoints.Groups.Admin.Applications.Reject, params); }

export async function createGroupAnnouncement(params: CreateGroupAnnouncementRequest) { return request<GroupAnnouncementItem>(Endpoints.Groups.Admin.Announcements.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function updateGroupAnnouncement(params: UpdateGroupAnnouncementRequest) { return request<GroupAnnouncementItem>(Endpoints.Groups.Admin.Announcements.Update, params); }
export async function deleteGroupAnnouncement(params: DeleteGroupAnnouncementRequest) { return request<void>(Endpoints.Groups.Admin.Announcements.Delete, params); }

// Groups admin（守卫子树新增端点，spec: api_spec_group.md）
export async function getGroupSettings(groupId: string) {
  return request<GroupSettings>(Endpoints.Groups.Admin.GetSettings, { group_id: groupId });
}
export async function listAdminGroupMembers(params: ListAdminGroupMembersRequest) {
  return request<PaginatedData<AdminGroupMemberItem>>(Endpoints.Groups.Admin.Members.List, params);
}
// 群主/群管理员撤回任意成员消息（无时间窗）；撤回自己的消息走 recallGroupMessage
export async function recallGroupMessageAsAdmin(params: RecallGroupMessageAdminRequest) {
  return request<void>(Endpoints.Groups.Admin.Messages.Recall, params);
}
