import { BaseService } from './base.service'
import { GROUPS } from '~/config/api'
import type {
  PaginatedData,
  GroupSearchItem,
  GroupSettings,
  GroupSettingsData,
  JoinedGroupItem,
  GroupMembersRequest,
  GroupMemberItem,
  BannedMemberItem,
  GroupApplicationItem,
  GroupInviteItem,
  GroupOpenTheDoorRequest,
  // 搜索类型
  SearchGroupsByKeywordRequest,
  SearchGroupsByTagRequest,
  SearchGroupsByLocationRequest,
  // 群组管理类型
  CreateGroupChatRequest,
  CreateGroupChatData,
  MyGroupChatsRequest,
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
  BannedMembersRequest,
  RemoveMemberRequest,
  LeaveGroupRequest,
  // 申请管理类型
  GroupApplicationsRequest,
  ApproveApplicationRequest,
  RejectApplicationRequest,
  CancelApplicationRequest,
  TransferOwnershipRequest,
  // 邀请管理类型
  MyInvitesRequest,
  AcceptInviteRequest,
  RejectInviteRequest
} from '~/types/api'

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
    return this.publicPost<GroupSettingsData>(GROUPS.SETTINGS, request)
  }

  async openTheDoor(request: GroupOpenTheDoorRequest) {
    return this.publicPost<GroupSearchItem>(GROUPS.OPEN_THE_DOOR, request)
  }

  // ============ 群组管理 API (protected) ============

  async create(request: CreateGroupChatRequest) {
    return this.post<CreateGroupChatData>(GROUPS.CREATE, request)
  }

  async getJoined(request: MyGroupChatsRequest = {}) {
    return this.post<PaginatedData<JoinedGroupItem>>(GROUPS.ME.LIST, request)
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

  async getMembers(request: GroupMembersRequest) {
    return this.publicPost<PaginatedData<GroupMemberItem>>(GROUPS.MEMBERS.LIST, request)
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
    return this.post<void>(GROUPS.MEMBERS.BAN, request)
  }

  async unbanMember(request: UnbanMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.UNBAN, request)
  }

  async getBannedMembers(request: BannedMembersRequest) {
    return this.post<PaginatedData<BannedMemberItem>>(GROUPS.MEMBERS.BANNED, request)
  }

  async removeMember(request: RemoveMemberRequest) {
    return this.post<void>(GROUPS.MEMBERS.REMOVE_MEMBER, request)
  }

  // ============ 申请管理 API ============

  async getApplications(request: GroupApplicationsRequest) {
    return this.post<PaginatedData<GroupApplicationItem>>(GROUPS.APPLICATIONS.LIST, request)
  }

  async approveApplication(request: ApproveApplicationRequest) {
    return this.post<void>(GROUPS.APPLICATIONS.APPROVE, request)
  }

  async rejectApplication(request: RejectApplicationRequest) {
    return this.post<void>(GROUPS.APPLICATIONS.REJECT, request)
  }

  // ============ 当前用户群组相关 API（/me/ 路径）============

  async cancelApplication(request: CancelApplicationRequest) {
    return this.post<void>(GROUPS.ME.CANCEL_APPLICATION, request)
  }

  async getInvites(request: MyInvitesRequest = {}) {
    return this.post<PaginatedData<GroupInviteItem>>(GROUPS.ME.INVITES, request)
  }

  async acceptInvite(request: AcceptInviteRequest) {
    return this.post<void>(GROUPS.ME.ACCEPT_INVITE, request)
  }

  async rejectInvite(request: RejectInviteRequest) {
    return this.post<void>(GROUPS.ME.REJECT_INVITE, request)
  }

  async leaveGroup(request: LeaveGroupRequest) {
    return this.post<void>(GROUPS.ME.LEAVE, request)
  }
}

export const groupsService = new GroupsService()
export function useGroupsService() { return groupsService }
