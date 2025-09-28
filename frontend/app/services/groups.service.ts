/**
 * 群组服务 - 现代化实现
 * 提供完整的群组管理功能
 */

import { BaseService } from './base.service'
import type { 
  GroupResponse, 
  GroupMemberResponse,
  PaginatedResponse,
  ConvertToPublicGroupRequest,
  ConvertToPublicGroupResponse
} from '~/types/api'

export interface CreateGroupRequest {
  name: string
  description?: string
  avatar_url?: string
  group_type: 'public_chat' | 'private_chat'
  location_name?: string
  latitude?: number
  longitude?: number
  max_members?: number
  join_approval_required?: boolean
  password?: string
  tags?: string[]
}

export interface UpdateGroupRequest {
  name?: string
  description?: string
  avatar_url?: string
  location_name?: string
  latitude?: number
  longitude?: number
  max_members?: number
  join_approval_required?: boolean
  password?: string
  tags?: string[]
}

export interface JoinGroupRequest {
  group_id: string
  password?: string
  message?: string
}

export class GroupsService extends BaseService {
  protected readonly baseUrl = '/protected/group'

  /**
   * 搜索群组
   */
  async searchGroups(params: {
    query?: string
    latitude?: number
    longitude?: number
    radius?: number
    group_type?: string
    tags?: string[]
    page?: number
    limit?: number
  }): Promise<PaginatedResponse<GroupResponse>> {
    // 使用公开接口搜索
    const http = useHttp()
    return http.get<PaginatedResponse<GroupResponse>>('/public/group/search', {
      query: params
    })
  }

  /**
   * 获取附近群组
   */
  async getNearbyGroups(
    latitude: number,
    longitude: number,
    radius: number = 5000,
    params?: {
      cursor?: string
      limit?: number
    }
  ): Promise<PaginatedResponse<GroupResponse>> {
    const http = useHttp()
    return http.get<PaginatedResponse<GroupResponse>>('/public/group/nearby', {
      query: {
        latitude,
        longitude,
        radius,
        ...params
      }
    })
  }

  /**
   * 获取我的群组
   */
  async getMyGroups(): Promise<PaginatedResponse<GroupResponse>> {
    return this.get<PaginatedResponse<GroupResponse>>('/my-groups')
  }

  /**
   * 获取群组详情
   */
  async getGroupDetails(groupId: string): Promise<GroupResponse> {
    // 使用公开接口获取详情
    const http = useHttp()
    return http.get<GroupResponse>(`/public/group/details/${groupId}`)
  }

  /**
   * 创建群组
   */
  async createGroup(data: CreateGroupRequest): Promise<{ group_id: string }> {
    return this.post<{ group_id: string }>('/create', data)
  }

  /**
   * 更新群组信息
   */
  async updateGroup(groupId: string, data: UpdateGroupRequest): Promise<GroupResponse> {
    return this.put<GroupResponse>(`/settings/${groupId}`, data)
  }

  /**
   * 上传群组头像
   */
  async uploadAvatar(groupId: string, file: File): Promise<{ avatar_url: string }> {
    return this.upload<{ avatar_url: string }>(`/avatar/${groupId}`, file)
  }

  /**
   * 加入群组
   */
  async joinGroup(data: JoinGroupRequest): Promise<void> {
    return this.post<void>('/join', data)
  }

  /**
   * 退出群组
   */
  async leaveGroup(groupId: string): Promise<void> {
    return this.post<void>('/leave', { group_id: groupId })
  }

  /**
   * 解散群组
   */
  async deleteGroup(groupId: string): Promise<void> {
    return this.delete<void>(`/${groupId}`)
  }

  /**
   * 获取群组成员
   */
  async getGroupMembers(
    groupId: string,
    params?: {
      role?: string
      cursor?: string
      limit?: number
    }
  ): Promise<PaginatedResponse<GroupMemberResponse>> {
    return this.paginate<GroupMemberResponse>(`/members/${groupId}`, params)
  }

  /**
   * 邀请成员
   */
  async inviteMembers(groupId: string, userIds: string[]): Promise<void> {
    return this.post<void>('/invite-members', {
      group_id: groupId,
      user_ids: userIds
    })
  }

  /**
   * 移除成员
   */
  async removeMember(groupId: string, userId: string): Promise<void> {
    return this.post<void>('/remove-member', {
      group_id: groupId,
      user_id: userId
    })
  }

  /**
   * 设置成员角色
   */
  async setMemberRole(
    groupId: string, 
    userId: string, 
    role: 'owner' | 'admin' | 'member'
  ): Promise<void> {
    return this.post<void>('/set-role', {
      group_id: groupId,
      user_id: userId,
      role
    })
  }

  /**
   * 转让群主
   */
  async transferOwnership(groupId: string, newOwnerId: string): Promise<void> {
    return this.post<void>('/transfer-ownership', {
      group_id: groupId,
      new_owner_id: newOwnerId
    })
  }

  /**
   * 将私聊转换为公开群组
   */
  async convertToPublic(data: ConvertToPublicGroupRequest): Promise<ConvertToPublicGroupResponse> {
    return this.post<ConvertToPublicGroupResponse>('/convert_to_public', data)
  }

  /**
   * 处理加群申请
   */
  async handleJoinRequest(
    groupId: string, 
    userId: string, 
    action: 'approve' | 'reject',
    message?: string
  ): Promise<void> {
    return this.post<void>('/handle-join-request', {
      group_id: groupId,
      user_id: userId,
      action,
      message
    })
  }

  /**
   * 获取加群申请列表
   */
  async getJoinRequests(groupId: string): Promise<PaginatedResponse<{
    user_id: string
    user_name: string
    user_avatar?: string
    message?: string
    requested_at: string
  }>> {
    return this.get<PaginatedResponse<{
      user_id: string
      user_name: string
      user_avatar?: string
      message?: string
      requested_at: string
    }>>(`/join-requests/${groupId}`)
  }

  /**
   * 设置群组公告
   */
  async setAnnouncement(groupId: string, announcement: string): Promise<void> {
    return this.post<void>('/set-announcement', {
      group_id: groupId,
      announcement
    })
  }

  /**
   * 获取群组统计
   */
  async getGroupStats(groupId: string): Promise<{
    member_count: number
    message_count: number
    created_at: string
    last_activity: string
  }> {
    return this.get<{
      member_count: number
      message_count: number
      created_at: string
      last_activity: string
    }>(`/stats/${groupId}`)
  }

  /**
   * 搜索群组成员
   */
  async searchMembers(
    groupId: string, 
    query: string
  ): Promise<GroupMemberResponse[]> {
    const response = await this.get<PaginatedResponse<GroupMemberResponse>>(`/search-members/${groupId}`, {
      query: { q: query }
    })
    return response.items
  }

  /**
   * 批量操作成员
   */
  async batchMemberOperation(
    groupId: string,
    operation: 'remove' | 'promote' | 'demote',
    userIds: string[]
  ): Promise<void> {
    return this.post<void>('/batch-member-operation', {
      group_id: groupId,
      operation,
      user_ids: userIds
    })
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
