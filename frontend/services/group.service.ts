/**
 * 群组服务层
 * 处理群组相关的API调用，与后端群组API对接
 * 使用 Composition API 格式，提高代码复用性
 */

import { useApiService } from './api.factory'
import type {
  CreateGroupRequest,
  CreateGroupResponse,
  GetGroupRequest,
  GetGroupResponse,
  JoinGroupRequest,
  JoinGroupResponse,
  LeaveGroupRequest,
  LeaveGroupResponse,
  KickMemberRequest,
  KickMemberResponse,
  GroupMembersRequest,
  GroupMembersResponse,
  UpdateGroupRequest,
  UpdateGroupResponse,
  SearchGroupByLocationRequest,
  SearchGroupByLocationResponse,
  SearchGroupByNameRequest,
  SearchGroupByNameResponse,
  SearchGroupByIdRequest,
  SearchGroupByIdResponse,
  GetUserGroupsRequest,
  GetUserGroupsResponse,
  GetOwnedGroupsRequest,
  GetOwnedGroupsResponse,
  TransferOwnershipRequest,
  TransferOwnershipResponse
} from '~/types/api/group'

/**
 * 群组服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useGroupService() {
  const apiService = useApiService()

  /**
   * 创建群组
   */
  const createGroup = async (data: CreateGroupRequest): Promise<CreateGroupResponse> => {
    const result = await apiService.safeRequest<CreateGroupResponse>('/group/create', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 更新群组 (暂时未实现)
   */
  const updateGroup = async (data: UpdateGroupRequest): Promise<UpdateGroupResponse> => {
    // TODO: 后端需要实现 update 端点
    console.warn('Group update not implemented on backend')
    return Promise.resolve({ code: 0, msg: 'Not implemented', data: null } as any)
  }

  /**
   * 加入群组
   */
  const joinGroup = async (data: JoinGroupRequest): Promise<JoinGroupResponse> => {
    const result = await apiService.safeRequest<JoinGroupResponse>('/group/join', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 离开群组
   */
  const leaveGroup = async (data: LeaveGroupRequest): Promise<LeaveGroupResponse> => {
    const result = await apiService.safeRequest<LeaveGroupResponse>('/group/leave', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 踢出群成员
   */
  const kickGroupMember = async (data: KickMemberRequest): Promise<KickMemberResponse> => {
    const result = await apiService.safeRequest<KickMemberResponse>('/group/kick', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取群组成员
   */
  const getGroupMembers = async (params: GroupMembersRequest): Promise<GroupMembersResponse> => {
    const result = await apiService.safeRequest<GroupMembersResponse>('/group/members', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 转让群主
   */
  const transferGroupOwnership = async (data: TransferOwnershipRequest): Promise<TransferOwnershipResponse> => {
    const result = await apiService.safeRequest<TransferOwnershipResponse>('/group/transfer_ownership', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按位置搜索群组
   */
  const searchGroupsByLocation = async (params: SearchGroupByLocationRequest): Promise<SearchGroupByLocationResponse> => {
    const result = await apiService.safeRequest<SearchGroupByLocationResponse>('/group/search_by_location', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按名称搜索群组
   */
  const searchGroupsByName = async (params: SearchGroupByNameRequest): Promise<SearchGroupByNameResponse> => {
    const result = await apiService.safeRequest<SearchGroupByNameResponse>('/group/search_by_name', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按ID搜索群组
   */
  const searchGroupsById = async (params: SearchGroupByIdRequest): Promise<SearchGroupByIdResponse> => {
    const result = await apiService.safeRequest<SearchGroupByIdResponse>('/group/search_by_id', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取用户加入的群组
   */
  const getUserGroups = async (params: GetUserGroupsRequest): Promise<GetUserGroupsResponse> => {
    const result = await apiService.safeRequest<GetUserGroupsResponse>('/group/user_groups', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取用户拥有的群组
   */
  const getOwnedGroups = async (params: GetOwnedGroupsRequest): Promise<GetOwnedGroupsResponse> => {
    const result = await apiService.safeRequest<GetOwnedGroupsResponse>('/group/owned_groups', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  return {
    createGroup,
    updateGroup,
    joinGroup,
    leaveGroup,
    kickGroupMember,
    getGroupMembers,
    transferGroupOwnership,
    searchGroupsByLocation,
    searchGroupsByName,
    searchGroupsById,
    getUserGroups,
    getOwnedGroups
  }
}