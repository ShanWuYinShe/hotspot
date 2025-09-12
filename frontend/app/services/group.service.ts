/**
 * 群组服务 - Composition API 风格
 * 提供群组相关的所有 API 调用功能
 */


import { GROUP_ENDPOINTS, buildGroupUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";
import type { 
  CreateGroupRequest,
  CreateGroupResponse,
  UpdateGroupRequest,
  UpdateGroupResponse,
  DeleteGroupRequest,
  DeleteGroupResponse,
  InviteUserRequest,
  InviteUserResponse,
  JoinGroupResponse,
  LeaveGroupResponse,
  KickMemberResponse,
  TransferOwnershipResponse,
  GroupInfo,
  GroupMembersResponse,
  SearchGroupByLocationResponse,
  SearchGroupByNameResponse,
  GetUserGroupsResponse,
  GetOwnedGroupsResponse
} from "~/types";
import { 
  validateGroupId, 
  validateUserId, 
  validateContent,
  validateCoordinates,
  validateRadius,
  validatePaginationParams,
  validateSearchKeyword
} from "~/utils/api-validation";
import { extractApiData, extractPaginatedData } from "~/utils/api-response";

export const useGroupService = () => {
  const api = useApi();

  // 基础群组操作
  const createGroup = async (data: CreateGroupRequest): Promise<CreateGroupResponse> => {
    // 发送前验证参数
    const validName = validateContent(data.name, 2, 32, '群组名称');
    const validDescription = data.description ? validateContent(data.description, 0, 300, '群组描述') : undefined;
    validateCoordinates(data.latitude, data.longitude);
    const validLocationName = validateContent(data.location_name, 1, 120, '位置名称');
    
    const response = await api.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.CREATE), {
      name: validName,
      description: validDescription,
      password: data.password,
      latitude: data.latitude,
      longitude: data.longitude,
      location_name: validLocationName
    }, { auth: true });
    
    return extractApiData(response);
  };

  const joinGroup = async (groupId: string, password?: string): Promise<JoinGroupResponse> => {
    // 发送前验证参数
    const validGroupId = validateGroupId(groupId);
    
    const response = await api.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.JOIN), { 
      group_id: validGroupId, 
      password 
    }, { auth: true });
    
    return extractApiData(response);
  };

  const leaveGroup = async (groupId: string): Promise<LeaveGroupResponse> => {
    // 发送前验证参数
    const validGroupId = validateGroupId(groupId);
    
    const response = await api.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.LEAVE), { 
      group_id: validGroupId 
    }, { auth: true });
    
    return extractApiData(response);
  };

  const getGroupDetails = async (groupId: string): Promise<GroupInfo> => {
    // 发送前验证参数
    const validGroupId = validateGroupId(groupId);
    
    const response = await api.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true), { 
      group_id: validGroupId 
    });
    
    return extractApiData(response);
  };

  // 更新群组（暂未实现 - hotspot.md 中未定义此端点）
  const updateGroup = async (data: UpdateGroupRequest): Promise<UpdateGroupResponse> => {
    throw new Error('群组更新功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.3.1 中只定义了9个群组端点，不包含更新功能
  };

  // 删除群组（暂未实现 - hotspot.md 中未定义此端点）
  const deleteGroup = async (data: DeleteGroupRequest): Promise<DeleteGroupResponse> => {
    throw new Error('群组删除功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.3.1 中只定义了9个群组端点，不包含删除功能
  };

  // 邀请成员（暂未实现 - hotspot.md 中未定义此端点）
  const inviteUser = async (groupId: string, userId: string): Promise<InviteUserResponse> => {
    throw new Error('群组邀请功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.3.1 中只定义了9个群组端点，不包含邀请功能
  };

  // 搜索和发现
  const searchNearbyGroups = async (
    latitude: number,
    longitude: number,
    radiusMeters?: number,
    cursor?: number
  ): Promise<SearchGroupByLocationResponse> => {
    // 发送前验证参数
    validateCoordinates(latitude, longitude);
    const validRadius = validateRadius(radiusMeters ?? DEFAULTS.nearbyRadiusMeters);
    const validPagination = validatePaginationParams({ cursor: cursor ? String(cursor) : undefined, limit: DEFAULTS.pageSize });
    
    const response = await api.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), {
      latitude,
      longitude,
      radius: validRadius,
      cursor: validPagination.cursor, // 🔥 修复：首次加载时不传递cursor，让后端处理默认值
      limit: validPagination.limit,
    });
    
    return extractPaginatedData(response);
  };

  const searchGroupsByName = async (query: string, cursor?: number): Promise<SearchGroupByNameResponse> => {
    // 发送前验证参数
    const validKeyword = validateSearchKeyword(query);
    const validPagination = validatePaginationParams({ cursor: cursor ? String(cursor) : undefined, limit: DEFAULTS.pageSize });
    
    const response = await api.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.SEARCH_BY_NAME, true), {
      keyword: validKeyword,
      cursor: validPagination.cursor || 0,
      limit: validPagination.limit,
    });
    
    return extractPaginatedData(response);
  };

  // 成员管理（公开访问，无需认证）
  const getGroupMembers = async (groupId: string, cursor?: string): Promise<GroupMembersResponse> => {
    // 发送前验证参数
    const validGroupId = validateGroupId(groupId);
    const validPagination = { cursor: cursor ? parseInt(cursor) : 0, limit: DEFAULTS.pageSize };
    
    const response = await api.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.MEMBERS, true), {
      group_id: validGroupId,
      cursor: validPagination.cursor,
      limit: validPagination.limit,
    });
    
    return extractPaginatedData(response);
  };

  const kickMember = async (groupId: string, userId: string): Promise<KickMemberResponse> => {
    // 发送前验证参数
    const validGroupId = validateGroupId(groupId);
    const validUserId = validateUserId(userId);
    
    const response = await api.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.KICK), { 
      group_id: validGroupId, 
      user_id: validUserId 
    }, { auth: true });
    
    return extractApiData(response);
  };

  const transferOwnership = async (groupId: string, newOwnerId: string): Promise<TransferOwnershipResponse> => {
    // 发送前验证参数
    const validGroupId = validateGroupId(groupId);
    const validUserId = validateUserId(newOwnerId);
    
    const response = await api.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.TRANSFER_OWNERSHIP), { 
      group_id: validGroupId, 
      new_owner_id: validUserId 
    }, { auth: true });
    
    return extractApiData(response);
  };

  // 获取我的群组（暂未实现 - hotspot.md 中未定义此端点）
  const getMyGroups = async (cursor?: number): Promise<GetUserGroupsResponse> => {
    throw new Error('获取我的群组功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.3.1 中只定义了9个群组端点，不包含此功能
  };

  // 获取拥有的群组（暂未实现 - hotspot.md 中未定义此端点）
  const getOwnedGroups = async (cursor?: number): Promise<GetOwnedGroupsResponse> => {
    throw new Error('获取拥有的群组功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.3.1 中只定义了9个群组端点，不包含此功能
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
    kickMember,
    transferOwnership,
    
    // 用户群组关系
    getMyGroups,
    getOwnedGroups,
  };
};
