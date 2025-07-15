/**
 * 群组相关API类型定义
 * 与后端 group API 严格对应
 */

import type { PaginatedResponse, Pagination } from '~/types/api'

// ==================== 群组信息 ====================

export interface GroupInfo {
  group_id: string // UUID
  name: string
  description?: string
  owner_id: number
  creator_name: string
  created_at: string
  last_active_at: string
  latitude: number
  longitude: number
  member_count: number
  distance: number
  location_name: string
  is_password_required: boolean
  is_member: boolean
  user_role: string
}

export interface GroupMemberInfo {
  id: number
  user_id: number
  nickname: string
  last_active_at: string
  role: string
  join_time: string
}

// ==================== 创建群组 ====================

export interface CreateGroupRequest {
  name: string
  description?: string
  password?: string
  latitude: number
  longitude: number
  location_name: string
}

export interface CreateGroupResponse {
  group_id: string
}

// ==================== 加入/离开群组 ====================

export interface JoinGroupRequest {
  group_id: string
  password?: string
}

export interface JoinGroupResponse {
  success: boolean
  role: string
}

export interface LeaveGroupRequest {
  group_id: string
}

export interface LeaveGroupResponse {
  success: boolean
}

// ==================== 群组成员管理 ====================

export interface KickMemberRequest {
  group_id: string
  user_id: number
}

export interface KickMemberResponse {
  success: boolean
}

export interface GroupMembersRequest {
  group_id: string
  cursor?: number
  limit?: number
}

export type GroupMembersResponse = PaginatedResponse<GroupMemberInfo>

// ==================== 群组更新 ====================

export interface UpdateGroupRequest {
  group_id: string
  name?: string
  description?: string
  avatar_url?: string
}

export interface UpdateGroupResponse {
  success: boolean
}

export interface GetGroupRequest {
  group_id: string
}

export interface GetGroupResponse {
  group: GroupInfo
}

// ==================== 搜索群组 ====================

export interface SearchGroupByLocationRequest {
  latitude: number
  longitude: number
  radius: number
  cursor?: number
  limit?: number
}

export interface SearchGroupByNameRequest {
  keyword: string
  cursor?: number
  limit?: number
}

export interface SearchGroupByIdRequest {
  group_id: string
}

export type SearchGroupByLocationResponse = PaginatedResponse<GroupInfo>
export type SearchGroupByNameResponse = PaginatedResponse<GroupInfo>
export type SearchGroupByIdResponse = GroupInfo

// ==================== 用户群组 ====================

export interface GetUserGroupsRequest {
  limit?: number
  offset?: number
}

export interface GetOwnedGroupsRequest {
  limit?: number
  offset?: number
}

export type GetUserGroupsResponse = PaginatedResponse<GroupInfo>
export type GetOwnedGroupsResponse = PaginatedResponse<GroupInfo>

// ==================== 群组权限 ====================

export interface TransferOwnershipRequest {
  group_id: string
  new_owner_id: number
}

export interface TransferOwnershipResponse {
  success: boolean
}

// ==================== 群组角色相关 ====================

// 群组角色枚举
export enum GroupRole {
  Owner = 'owner',
  Admin = 'admin',
  Member = 'member'
}
