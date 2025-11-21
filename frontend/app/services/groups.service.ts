/**
 * 群组服务 - 符合API规范的实现
 * 所有API操作统一使用POST方法
 * 基于 api_spec.mdc 第3.6节和第3.7节 群组管理API规范
 */

import { BaseService } from './base.service'
import type { GroupResponse, GroupMemberResponseData, PaginatedResponse } from '~/types/api'

export interface DiscoverGroupsRequest {
  keyword?: string
  latitude?: number
  longitude?: number
  radius?: number
  limit?: number
  cursor?: string
}

export interface CreateGroupRequest {
  name: string
  description?: string
  avatar_url?: string
  settings: {
    join_method: 'open' | 'approval' | 'invite_only'
    is_searchable: boolean
    max_members?: number
    password_hash?: string
  }
}

export interface GetJoinedRequest {
  limit?: number
  cursor?: string
}

export interface OpenTheDoorRequest {
  id: string
}

export interface UpdateGroupRequest {
  group_chat_id: string
  name?: string
  description?: string
  avatar_url?: string
  settings?: {
    join_method?: 'open' | 'approval' | 'invite_only'
    is_searchable?: boolean
    max_members?: number
    password_hash?: string
  }
}

export interface DeleteGroupRequest {
  group_chat_id: string
}

export interface GetMembersRequest {
  group_chat_id: string
  role?: 'owner' | 'admin' | 'member'
  limit?: number
  cursor?: string
}

export interface InviteMemberRequest {
  group_chat_id: string
  user_id: string
  message?: string
}

export interface JoinGroupRequest {
  group_chat_id: string
  password_hash?: string
}

export interface ApplyToGroupRequest {
  group_chat_id: string
  message?: string
}

export interface SetMemberRoleRequest {
  group_chat_id: string
  user_id: string
  role: 'admin' | 'member'
}

export interface MuteMemberRequest {
  group_chat_id: string
  user_id: string
  duration_minutes: number
}

export interface RemoveMemberRequest {
  group_chat_id: string
  user_id: string
}

export interface LeaveGroupRequest {
  group_chat_id: string
}

export interface GetApplicationsRequest {
  group_chat_id: string
  status?: 'pending' | 'accepted' | 'rejected' | 'all'
  limit?: number
  cursor?: string
}

export interface ApproveApplicationRequest {
  application_id: string
}

export interface RejectApplicationRequest {
  application_id: string
  reason?: string
}

export interface TransferOwnershipRequest {
  group_chat_id: string
  new_owner_id: string
}

export interface GetInvitesRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'all'
  limit?: number
  cursor?: string
}

export interface AcceptInviteRequest {
  invite_id: string
}

export interface RejectInviteRequest {
  invite_id: string
}

export class GroupsService extends BaseService {
  protected readonly baseUrl = '/api/v1'

  /**
   * 发现群组（公开接口）
   * POST /api/v1/public/group_chats/discover
   */
  async discoverGroups(request: DiscoverGroupsRequest): Promise<PaginatedResponse<GroupResponse>> {
    return this.publicPost<PaginatedResponse<GroupResponse>>('/public/group_chats/discover', request)
  }

  /**
   * 创建群组
   * POST /api/v1/protected/group_chats/create
   */
  async createGroup(request: CreateGroupRequest): Promise<{ group_chat_id: string }> {
    return this.post<{ group_chat_id: string }>('/protected/group_chats/create', request)
  }

  /**
   * 获取我的群组
   * POST /api/v1/protected/group_chats/get_joined
   */
  async getJoined(request: GetJoinedRequest = {}): Promise<PaginatedResponse<GroupResponse>> {
    return this.post<PaginatedResponse<GroupResponse>>('/protected/group_chats/get_joined', request)
  }

  /**
   * 获取群组详情（公开接口）
   * POST /api/v1/public/group_chats/open_the_door
   */
  async openTheDoor(request: OpenTheDoorRequest): Promise<GroupResponse> {
    return this.publicPost<GroupResponse>('/public/group_chats/open_the_door', request)
  }

  /**
   * 更新群组资料
   * POST /api/v1/protected/group_chats/update
   */
  async updateGroup(request: UpdateGroupRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/update', request)
  }

  /**
   * 解散群组
   * POST /api/v1/protected/group_chats/delete
   */
  async deleteGroup(request: DeleteGroupRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/delete', request)
  }

  /**
   * 获取成员列表（公开接口）
   * POST /api/v1/public/group_chats/members/get_list
   */
  async getMembers(request: GetMembersRequest): Promise<PaginatedResponse<GroupMemberResponseData>> {
    return this.publicPost<PaginatedResponse<GroupMemberResponseData>>('/public/group_chats/members/get_list', request)
  }

  /**
   * 邀请成员
   * POST /api/v1/protected/group_chats/members/invite
   */
  async inviteMember(request: InviteMemberRequest): Promise<{ id: string; expires_at: string }> {
    return this.post<{ id: string; expires_at: string }>('/protected/group_chats/members/invite', request)
  }

  /**
   * 加入群组
   * POST /api/v1/protected/group_chats/members/join
   */
  async joinGroup(request: JoinGroupRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/members/join', request)
  }

  /**
   * 申请加入
   * POST /api/v1/protected/group_chats/members/apply
   */
  async applyToGroup(request: ApplyToGroupRequest): Promise<{ id: string }> {
    return this.post<{ id: string }>('/protected/group_chats/members/apply', request)
  }

  /**
   * 设置成员角色
   * POST /api/v1/protected/group_chats/members/set_role
   */
  async setMemberRole(request: SetMemberRoleRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/members/set_role', request)
  }

  /**
   * 禁言成员
   * POST /api/v1/protected/group_chats/members/mute
   */
  async muteMember(request: MuteMemberRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/members/mute', request)
  }

  /**
   * 移除成员
   * POST /api/v1/protected/group_chats/members/remove
   */
  async removeMember(request: RemoveMemberRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/members/remove', request)
  }

  /**
   * 退出群组
   * POST /api/v1/protected/group_chats/members/leave
   */
  async leaveGroup(request: LeaveGroupRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/members/leave', request)
  }

  /**
   * 获取群组申请列表
   * POST /api/v1/protected/group_chats/applications/get_list
   */
  async getApplications(request: GetApplicationsRequest): Promise<PaginatedResponse<{
    id: string
    applicant: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    message?: string
    status: 'pending' | 'accepted' | 'rejected'
    created_at: string
    handled_at?: string
  }>> {
    return this.post('/protected/group_chats/applications/get_list', request)
  }

  /**
   * 审批群组申请
   * POST /api/v1/protected/group_chats/applications/approve
   */
  async approveApplication(request: ApproveApplicationRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/applications/approve', request)
  }

  /**
   * 拒绝群组申请
   * POST /api/v1/protected/group_chats/applications/reject
   */
  async rejectApplication(request: RejectApplicationRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/applications/reject', request)
  }

  /**
   * 转让群主
   * POST /api/v1/protected/group_chats/transfer_ownership
   */
  async transferOwnership(request: TransferOwnershipRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/transfer_ownership', request)
  }

  /**
   * 获取我的群组邀请列表
   * POST /api/v1/protected/group_chats/invites/get_list
   */
  async getInvites(request: GetInvitesRequest = {}): Promise<PaginatedResponse<{
    id: string
    group: {
      group_chat_id: string
      name: string
      avatar_url?: string
      member_count: number
    }
    inviter: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    message?: string
    status: 'pending' | 'accepted' | 'rejected'
    created_at: string
    expires_at?: string
  }>> {
    return this.post('/protected/group_chats/invites/get_list', request)
  }

  /**
   * 接受群组邀请
   * POST /api/v1/protected/group_chats/invites/accept
   */
  async acceptInvite(request: AcceptInviteRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/invites/accept', request)
  }

  /**
   * 拒绝群组邀请
   * POST /api/v1/protected/group_chats/invites/reject
   */
  async rejectInvite(request: RejectInviteRequest): Promise<void> {
    return this.post<void>('/protected/group_chats/invites/reject', request)
  }

  /**
   * 上传群组头像（使用FormData）
   */
  async uploadAvatar(groupId: string, file: File): Promise<{ avatar_url: string }> {
    return this.upload<{ avatar_url: string }>('/protected/media/upload_file', file, {
      category: 'group_image',
      group_id: groupId
    })
  }

  /**
   * 获取附近群组（别名方法，调用 discoverGroups）
   */
  async getNearbyGroups(request: DiscoverGroupsRequest): Promise<PaginatedResponse<GroupResponse>> {
    return this.discoverGroups(request)
  }

  /**
   * 搜索群组（别名方法，调用 discoverGroups）
   */
  async searchGroups(request: DiscoverGroupsRequest): Promise<PaginatedResponse<GroupResponse>> {
    return this.discoverGroups(request)
  }
}

// 创建服务实例
export const groupsService = new GroupsService()

// 提供 composable 接口
export function useGroupsService() {
  return groupsService
}

// 兼容旧的导出名称
export function useGroupService() {
  return groupsService
}
