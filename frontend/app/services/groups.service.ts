import { BaseService } from './base.service'
import { GROUPS } from '~/config/api'
import type {
  PaginatedData,
  GroupSearchItem,
  GroupSettings,
  JoinedGroupItem,
  GetGroupMembersRequest,
  GroupMemberItem,
  BannedMemberItem,
  GroupApplicationItem,
  GroupInviteItem,
  // 搜索类型
  SearchGroupsByKeywordRequest,
  SearchGroupsByTagRequest,
  SearchGroupsByLocationRequest,
  // 群组管理类型
  CreateGroupChatRequest,
  CreateGroupChatData,
  GetJoinedGroupChatsRequest,
  UpdateGroupChatRequest,
  DeleteGroupChatRequest,
  // 成员管理类型
  InviteMemberRequest,
  InviteMemberData,
  JoinGroupRequest,
  ApplyToGroupRequest,
  ApplyToGroupData,
  SetMemberRoleRequest,
  MuteMemberRequest,
  UnmuteMemberRequest,
  BanMemberRequest,
  UnbanMemberRequest,
  GetBannedMembersRequest,
  RemoveMemberRequest,
  LeaveGroupRequest,
  // 申请管理类型
  GetGroupApplicationsRequest,
  ApproveGroupApplicationRequest,
  RejectGroupApplicationRequest,
  CancelGroupApplicationRequest,
  TransferOwnershipRequest,
  // 邀请管理类型
  GetGroupInvitesRequest,
  AcceptGroupInviteRequest,
  RejectGroupInviteRequest
} from '~/types/api'

// 群组设置响应类型
interface GroupSettingsData {
  id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
}

class GroupsService extends BaseService {
  // ============ 搜索 API (public) ============

  async searchByKeyword(request: SearchGroupsByKeywordRequest) {
    return this.publicPost<PaginatedData<GroupSearchItem>>(GROUPS.SEARCH_BY_KEYWORD, request)
  }

  async searchByTag(request: SearchGroupsByTagRequest) {
    return this.publicPost<PaginatedData<GroupSearchItem>>(GROUPS.SEARCH_BY_TAG, request)
  }

  async searchByLocation(request: SearchGroupsByLocationRequest) {
    return this.publicPost<PaginatedData<GroupSearchItem>>(GROUPS.SEARCH_BY_LOCATION, request)
  }

  async getSettings(request: { id: string }) {
    return this.publicPost<GroupSettingsData>(GROUPS.GET_SETTINGS, request)
  }

  async openTheDoor(request: { id: string; latitude: number; longitude: number }) {
    return this.publicPost<GroupSearchItem>(GROUPS.OPEN_THE_DOOR, request)
  }

  // ============ 群组管理 API (protected) ============

  async create(request: CreateGroupChatRequest) {
    return this.post<CreateGroupChatData>(GROUPS.CREATE, request)
  }

  async getJoined(request: GetJoinedGroupChatsRequest = {}) {
    return this.post<PaginatedData<JoinedGroupItem>>(GROUPS.ME.GET_JOINED, request)
  }

  async update(request: UpdateGroupChatRequest) {
    return this.post<void>(GROUPS.UPDATE, request)
  }

  async deleteGroup(request: DeleteGroupChatRequest) {
    return this.post<void>(GROUPS.DELETE, request)
  }

  async transferOwnership(request: TransferOwnershipRequest) {
    return this.post<void>(GROUPS.TRANSFER_OWNERSHIP, request)
  }

  // ============ 成员管理 API ============

  async getMembers(request: GetGroupMembersRequest) {
    return this.publicPost<PaginatedData<GroupMemberItem>>(GROUPS.MEMBERS.GET_LIST, request)
  }

  async inviteMember(request: InviteMemberRequest) {
    return this.post<InviteMemberData>(GROUPS.MEMBERS.INVITE, request)
  }

  async joinGroup(request: JoinGroupRequest) {
    return this.post<void>(GROUPS.MEMBERS.JOIN, request)
  }

  async applyToGroup(request: ApplyToGroupRequest) {
    return this.post<ApplyToGroupData>(GROUPS.MEMBERS.APPLY, request)
  }

  async setMemberRole(request: SetMemberRoleRequest) {
    return this.post<void>(GROUPS.MEMBERS.SET_ROLE, request)
  }

  async muteMember(request: MuteMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.MUTE, request)
  }

  async unmuteMember(request: UnmuteMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.UNMUTE, request)
  }

  async banMember(request: BanMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.BAN_MEMBER, request)
  }

  async unbanMember(request: UnbanMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.UNBAN_MEMBER, request)
  }

  async getBannedMembers(request: GetBannedMembersRequest) {
    return this.post<PaginatedData<BannedMemberItem>>(GROUPS.MEMBERS.GET_BANNED_LIST, request)
  }

  async removeMember(request: RemoveMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.REMOVE_MEMBER, request)
  }

  // ============ 申请管理 API ============

  async getApplications(request: GetGroupApplicationsRequest) {
    return this.post<PaginatedData<GroupApplicationItem>>(GROUPS.APPLICATIONS.GET_LIST, request)
  }

  async approveApplication(request: ApproveGroupApplicationRequest) {
    return this.post<void>(GROUPS.APPLICATIONS.APPROVE, request)
  }

  async rejectApplication(request: RejectGroupApplicationRequest) {
    return this.post<void>(GROUPS.APPLICATIONS.REJECT, request)
  }

  // ============ 当前用户群组相关 API（/me/ 路径）============

  async cancelApplication(request: CancelGroupApplicationRequest) {
    return this.post<void>(GROUPS.ME.CANCEL_APPLICATION, request)
  }

  async getInvites(request: GetGroupInvitesRequest = {}) {
    return this.post<PaginatedData<GroupInviteItem>>(GROUPS.ME.GET_INVITES_LIST, request)
  }

  async acceptInvite(request: AcceptGroupInviteRequest) {
    return this.post<void>(GROUPS.ME.ACCEPT_INVITE, request)
  }

  async rejectInvite(request: RejectGroupInviteRequest) {
    return this.post<void>(GROUPS.ME.REJECT_INVITE, request)
  }

  async leaveGroup(request: LeaveGroupRequest) {
    return this.post<void>(GROUPS.ME.LEAVE_GROUP, request)
  }
}

export const groupsService = new GroupsService()
export function useGroupsService() { return groupsService }
