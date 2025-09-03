/**
 * 群组服务 - 最简化版本  
 * 保持基本接口，供现有代码调用
 */


import { GROUP_ENDPOINTS, buildGroupUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";
import type { 
  CreateGroupRequest,
  CreateGroupResponse,
  UpdateGroupRequest,
  UpdateGroupResponse,
  GroupInfo,
  GroupMembersResponse,
  SearchGroupByLocationResponse,
  SearchGroupByNameResponse,
  GetUserGroupsResponse,
  GetOwnedGroupsResponse
} from "~/types";

export const useGroupService = () => {
  const api = useApi();

  // 基础群组操作
  const createGroup = async (data: CreateGroupRequest): Promise<CreateGroupResponse> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.CREATE), data, { auth: true });
  };

  const joinGroup = async (groupId: string, password?: string): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.JOIN), { group_id: groupId, password }, { auth: true });
  };

  const leaveGroup = async (groupId: string): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.LEAVE), { group_id: groupId }, { auth: true });
  };

  const getGroupDetails = async (groupId: string): Promise<GroupInfo> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.SEARCH_BY_ID), { group_id: groupId }, { auth: true });
  };

  const updateGroup = async (data: any): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.UPDATE), data, { auth: true });
  };

  const deleteGroup = async (data: any): Promise<any> => {
    return await api.post(buildGroupUrl('/delete'), { group_id: data.group_id }, { auth: true });
  };

  // 新增：邀请成员
  const inviteUser = async (groupId: string, userId: string): Promise<any> => {
    return await api.post(buildGroupUrl(`${GROUP_ENDPOINTS.INVITE}/${groupId}`), { user_id: userId }, { auth: true });
  };

  // 搜索和发现
  const searchNearbyGroups = async (
    latitude: number,
    longitude: number,
    radiusMeters?: number,
    cursor?: number
  ): Promise<SearchGroupByLocationResponse> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.SEARCH_BY_LOCATION), {
      latitude,
      longitude,
      radius: radiusMeters ?? DEFAULTS.nearbyRadiusMeters,
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  const searchGroupsByName = async (query: string, cursor?: number): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.SEARCH_BY_NAME), {
      query,
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 成员管理
  const getGroupMembers = async (groupId: string, cursor?: number): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.MEMBERS), {
      group_id: groupId,
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  const removeGroupMember = async (data: any): Promise<any> => {
    return await api.post(buildGroupUrl(`/kick/${data.group_id}/${data.user_id}`), {}, { auth: true });
  };

  const kickMember = async (groupId: string, userId: number): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.KICK), { group_id: groupId, target_user_id: userId }, { auth: true });
  };

  const updateGroupMemberRole = async (data: any): Promise<any> => {
    return await api.post(buildGroupUrl(`/update_role/${data.group_id}/${data.user_id}`), { 
      role: data.role 
    }, { auth: true });
  };

  const transferOwnership = async (groupId: string, newOwnerId: number): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.TRANSFER_OWNERSHIP), { group_id: groupId, target_user_id: newOwnerId }, { auth: true });
  };

  // 用户群组关系
  const getMyGroups = async (cursor?: number): Promise<GetUserGroupsResponse> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.USER_GROUPS), {
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  const getOwnedGroups = async (cursor?: number): Promise<any> => {
    return await api.post(buildGroupUrl(GROUP_ENDPOINTS.OWNED_GROUPS), {
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  return {
    // 基础操作
    createGroup,
    joinGroup,
    leaveGroup,
    getGroupDetails,
    updateGroup,
    deleteGroup,
    inviteUser,
    
    // 搜索发现
    searchNearbyGroups,
    searchGroupsByName,
    
    // 成员管理
    getGroupMembers,
    removeGroupMember,
    kickMember,
    updateGroupMemberRole,
    transferOwnership,
    
    // 用户群组
    getMyGroups,
    getOwnedGroups,
  };
};
