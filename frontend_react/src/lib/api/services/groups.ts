/**
 * 群组服务
 * 群组相关 API
 * 
 * 基于 @api_spec_group.mdc 规范
 */

import { apiRequest, apiRequestPublic, apiRequestIdempotent } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  GroupCard,
  GroupInfo,
  GroupSearchItem,
  MyGroupState,
  GroupMessageItem,
  UserInfo,
  Location,
  GroupJoinMethod,
  GroupRole,
  SetableGroupRole,
  SendMessageType,
} from '@/types/api';

// ============== 群组搜索 ==============

interface SearchByKeywordParams {
  kw: string;
  limit?: number;
  cursor?: string;
}

interface SearchByTagParams {
  tag: string;
  limit?: number;
  cursor?: string;
}

interface SearchByLocationParams {
  kw?: string;
  radius: number;
  limit?: number;
  cursor?: string;
}

/**
 * 按关键词搜索群组
 * Token 可选；登录后可见更多群组
 * @api POST /api/v1/group_chats/search_by_keyword
 */
export async function searchByKeyword(
  params: SearchByKeywordParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.GROUP_CHATS.SEARCH_BY_KEYWORD,
    params
  );
}

/**
 * 按标签搜索群组
 * Token 可选；登录后可见更多群组
 * @api POST /api/v1/group_chats/search_by_tag
 */
export async function searchByTag(
  params: SearchByTagParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.GROUP_CHATS.SEARCH_BY_TAG,
    params
  );
}

/**
 * 按位置搜索群组
 * Token 可选；登录后可见更多群组
 * @api POST /api/v1/group_chats/search_by_location
 */
export async function searchByLocation(
  params: SearchByLocationParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.GROUP_CHATS.SEARCH_BY_LOCATION,
    params
  );
}

// ============== 群组详情 ==============

/**
 * 获取群组详情
 * Token 可选；登录后返回 my_state
 * @api POST /api/v1/group_chats/open_the_door
 */
export async function getGroup(
  groupChatId: string
): Promise<ApiResponse<GroupCard>> {
  return apiRequestPublic<GroupCard>(ENDPOINTS.GROUP_CHATS.OPEN_THE_DOOR, {
    group_chat_id: groupChatId,
  });
}

/**
 * 获取我在群组中的状态
 * @api POST /api/v1/group_chats/my_state
 */
export async function getGroupState(
  groupChatId: string
): Promise<ApiResponse<MyGroupState>> {
  return apiRequest<MyGroupState>(ENDPOINTS.GROUP_CHATS.MY_STATE, {
    group_chat_id: groupChatId,
  });
}

// ============== 我的群组 ==============

interface MyGroupsParams {
  limit?: number;
  cursor?: string;
}

interface GroupListItem extends GroupInfo {
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_message?: {
    message_id: string;
    content?: string;
    type: string;
    sender: UserInfo;
    sent_at: string;
  };
  last_activity_at: string;
}

/**
 * 获取我加入的群组列表
 * @api POST /api/v1/group_chats/me
 */
export async function getMyGroups(
  params: MyGroupsParams = {}
): Promise<ApiResponse<PaginatedData<GroupListItem>>> {
  return apiRequest<PaginatedData<GroupListItem>>(
    ENDPOINTS.GROUP_CHATS.ME,
    params
  );
}

// ============== 群组管理 ==============

interface CreateGroupParams {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  location: Location;
  join_method?: GroupJoinMethod;
  password?: string;
  is_searchable?: boolean;
  max_members?: number;
}

/**
 * 创建群组
 * @api POST /api/v1/group_chats/create
 */
export async function createGroup(
  params: CreateGroupParams
): Promise<ApiResponse<{ group_chat_id: string }>> {
  return apiRequestIdempotent<{ group_chat_id: string }>(
    ENDPOINTS.GROUP_CHATS.CREATE,
    params
  );
}

interface UpdateGroupParams {
  group_chat_id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  location?: Location;
  join_method?: GroupJoinMethod;
  password?: string;
  is_searchable?: boolean;
  max_members?: number;
}

/**
 * 更新群组信息
 * @api POST /api/v1/group_chats/update
 */
export async function updateGroup(
  params: UpdateGroupParams
): Promise<ApiResponse<GroupCard>> {
  return apiRequest<GroupCard>(ENDPOINTS.GROUP_CHATS.UPDATE, params);
}

interface UpdateMySettingsParams {
  group_chat_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

/**
 * 更新我在群组中的设置
 * @api POST /api/v1/group_chats/update_my_settings
 */
export async function updateMySettings(
  params: UpdateMySettingsParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.UPDATE_MY_SETTINGS, params);
}

/**
 * 删除群组
 * @api POST /api/v1/group_chats/delete
 */
export async function deleteGroup(
  groupChatId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.DELETE, {
    group_chat_id: groupChatId,
  });
}

// ============== 成员管理 ==============

interface MembersParams {
  group_chat_id: string;
  role?: GroupRole;
  limit?: number;
  cursor?: string;
}

interface GroupMember {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  is_muted: boolean;
  muted_until?: string;
}

/**
 * 获取群组成员列表
 * Token 可选
 * @api POST /api/v1/group_chats/members
 */
export async function getMembers(
  params: MembersParams
): Promise<ApiResponse<PaginatedData<GroupMember>>> {
  return apiRequestPublic<PaginatedData<GroupMember>>(
    ENDPOINTS.GROUP_CHATS.MEMBERS,
    params
  );
}

/**
 * 加入群组
 * @api POST /api/v1/group_chats/members/join
 */
export async function joinGroup(
  groupChatId: string,
  password?: string
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_JOIN, {
    group_chat_id: groupChatId,
    password,
  });
}

/**
 * 申请加入群组
 * @api POST /api/v1/group_chats/members/apply
 */
export async function applyToJoin(
  groupChatId: string,
  message?: string
): Promise<ApiResponse<{ application_id: string }>> {
  return apiRequestIdempotent<{ application_id: string }>(
    ENDPOINTS.GROUP_CHATS.MEMBERS_APPLY,
    {
      group_chat_id: groupChatId,
      message,
    }
  );
}

/**
 * 邀请用户加入群组
 * @api POST /api/v1/group_chats/members/invite
 */
export async function inviteMember(
  groupChatId: string,
  userId: number,
  message?: string
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_INVITE, {
    group_chat_id: groupChatId,
    user_id: userId,
    message,
  });
}

/**
 * 设置成员角色
 * @api POST /api/v1/group_chats/members/set_role
 */
export async function setMemberRole(
  groupChatId: string,
  userId: number,
  role: SetableGroupRole
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_SET_ROLE, {
    group_chat_id: groupChatId,
    user_id: userId,
    role,
  });
}

/**
 * 禁言成员
 * @api POST /api/v1/group_chats/members/mute
 */
export async function muteMember(
  groupChatId: string,
  userId: number,
  durationMinutes: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_MUTE, {
    group_chat_id: groupChatId,
    user_id: userId,
    duration_minutes: durationMinutes,
  });
}

/**
 * 解除禁言
 * @api POST /api/v1/group_chats/members/unmute
 */
export async function unmuteMember(
  groupChatId: string,
  userId: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_UNMUTE, {
    group_chat_id: groupChatId,
    user_id: userId,
  });
}

/**
 * 封禁成员
 * @api POST /api/v1/group_chats/members/ban
 */
export async function banMember(
  groupChatId: string,
  userId: number,
  reason?: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_BAN, {
    group_chat_id: groupChatId,
    user_id: userId,
    reason,
  });
}

/**
 * 解除封禁
 * @api POST /api/v1/group_chats/members/unban
 */
export async function unbanMember(
  groupChatId: string,
  userId: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_UNBAN, {
    group_chat_id: groupChatId,
    user_id: userId,
  });
}

/**
 * 移除成员
 * @api POST /api/v1/group_chats/members/remove_member
 */
export async function removeMember(
  groupChatId: string,
  userId: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.MEMBERS_REMOVE, {
    group_chat_id: groupChatId,
    user_id: userId,
  });
}

/**
 * 退出群组
 * @api POST /api/v1/group_chats/me/leave
 */
export async function leaveGroup(
  groupChatId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.GROUP_CHATS.ME_LEAVE, {
    group_chat_id: groupChatId,
  });
}

// ============== 群消息 ==============

interface SendGroupMessageParams {
  group_chat_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
  mentions?: number[];
}

/**
 * 发送群消息
 * @api POST /api/v1/group_messages/send
 */
export async function sendGroupMessage(
  params: SendGroupMessageParams
): Promise<ApiResponse<{ message_id: string; sent_at: string }>> {
  return apiRequestIdempotent<{ message_id: string; sent_at: string }>(
    ENDPOINTS.GROUP_MESSAGES.SEND,
    params
  );
}

interface GroupMessageHistoryParams {
  group_chat_id: string;
  include_system?: boolean;
  limit?: number;
  cursor?: string;
}

/**
 * 获取群消息历史
 * @api POST /api/v1/group_messages/history
 */
export async function getGroupMessageHistory(
  params: GroupMessageHistoryParams
): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
  return apiRequest<PaginatedData<GroupMessageItem>>(
    ENDPOINTS.GROUP_MESSAGES.HISTORY,
    params
  );
}
