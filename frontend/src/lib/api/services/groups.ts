/**
 * 群组服务
 */

import { post, paginate, generateIdempotencyKey, type RequestOptions } from '../client';
import { GROUPS, GROUP_MESSAGES } from '../endpoints';
import type {
  ApplicationStatus,
  GroupItem,
  GroupMemberItem,
  BannedMemberItem,
  GroupAnnouncementItem,
  GroupApplicationItem,
  GroupInviteItem,
  GroupMessageItem,
  GroupSentInviteItem,
  MessageSent,
  JoinGroupResult,
  Invitation,
  Location,
  SendMessageType,
  SettableGroupRole,
  PaginationParams,
  GeoPoint,
  GroupRole
} from '$lib/types/api';

export const groupsService = {
  // ==================== 搜索 ====================

  /**
   * 搜索群组（对齐 spec SearchGroupsRequest）
   */
  search(
    params: {
      keyword?: string;
      tag?: string;
      radius?: number;
      sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
      reference_point: GeoPoint;
      pagination?: PaginationParams;
    },
    options?: RequestOptions
  ) {
    return paginate<GroupItem>(GROUPS.SEARCH, params, options);
  },

  // ==================== 群组信息 ====================

  /**
   * 获取群组详情
   */
  getGroup(data: { group_id: string }, options?: RequestOptions) {
    return post<GroupItem>(GROUPS.OPEN_THE_DOOR, data, options);
  },

  // ==================== 群组操作 ====================

  /**
   * 创建群组
   */
  create(data: {
    name: string;
    location: Location;
    description?: string;
    avatar_file_id?: string;
    tags?: string[];
    password?: string;
    join_method?: 'open' | 'approval' | 'invite_only';
    is_searchable?: boolean;
    max_members?: number;
  }) {
    return post<GroupItem>(GROUPS.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 更新群组
   */
  update(data: {
    group_id: string;
    name?: string;
    description?: string;
    avatar_file_id?: string;
    tags?: string[];
    password?: string;
    clear_password?: boolean;
    join_method?: 'open' | 'approval' | 'invite_only';
    is_searchable?: boolean;
    max_members?: number;
    location?: Location;
  }) {
    return post<void>(GROUPS.UPDATE, data);
  },

  /**
   * 更新我的群组设置
   */
  updateMySettings(data: { group_id: string; is_muted?: boolean; is_pinned?: boolean }) {
    return post<void>(GROUPS.ME.UPDATE_SETTINGS, data);
  },

  /**
   * 解散群组（SHOULD 幂等）
   */
  dismiss(data: { group_id: string }) {
    return post<void>(GROUPS.DISMISS, data, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 转让群主（SHOULD 幂等）
   */
  transferOwnership(data: { group_id: string; new_owner_id: string }) {
    return post<void>(GROUPS.ME.TRANSFER_OWNERSHIP, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  // ==================== 成员管理 ====================

  /**
   * 获取成员列表
   */
  getMembers(params: { group_id: string; role?: GroupRole; pagination?: PaginationParams }) {
    return paginate<GroupMemberItem>(GROUPS.MEMBERS.LIST, params);
  },

  /**
   * 搜索成员
   */
  searchMembers(params: {
    group_id: string;
    keyword: string;
    role?: GroupRole;
    pagination?: PaginationParams;
  }) {
    return paginate<GroupMemberItem>(GROUPS.MEMBERS.SEARCH, params);
  },

  /**
   * 邀请成员（单个用户）
   */
  invite(data: { group_id: string; user_id: string; message?: string }) {
    return post<Invitation>(GROUPS.MEMBERS.INVITE, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 我发出的邀请列表
   */
  getSentInvites(params: {
    group_id: string;
    status?: ApplicationStatus | 'all';
    pagination?: PaginationParams;
  }) {
    return paginate<GroupSentInviteItem>(GROUPS.MEMBERS.INVITES_LIST, params);
  },

  /**
   * 撤销已发送的邀请
   */
  revokeSentInvite(data: { group_id: string; invite_id: string; reason?: string }) {
    return post<void>(GROUPS.MEMBERS.INVITES_REVOKE, data);
  },

  /**
   * 加入群组（自动根据 join_method 处理：open 直接加入，approval 发起申请）
   */
  join(data: { group_id: string; password?: string; message?: string }) {
    return post<JoinGroupResult>(GROUPS.ME.JOIN, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 设置成员角色
   */
  setRole(data: { group_id: string; user_id: string; role: SettableGroupRole }) {
    return post<void>(GROUPS.MEMBERS.SET_ROLE, data);
  },

  /**
   * 禁言成员
   */
  mute(data: { group_id: string; user_id: string; duration_minutes: number }) {
    return post<void>(GROUPS.MEMBERS.MUTE, data);
  },

  /**
   * 解除禁言
   */
  unmute(data: { group_id: string; user_id: string }) {
    return post<void>(GROUPS.MEMBERS.UNMUTE, data);
  },

  /**
   * 封禁成员
   */
  ban(data: { group_id: string; user_id: string; reason?: string }) {
    return post<void>(GROUPS.MEMBERS.BAN, data, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 解除封禁
   */
  unban(data: { group_id: string; user_id: string }) {
    return post<void>(GROUPS.MEMBERS.UNBAN, data);
  },

  /**
   * 获取封禁列表
   */
  getBanned(params: { group_id: string; pagination?: PaginationParams }) {
    return paginate<BannedMemberItem>(GROUPS.MEMBERS.LIST_BANNED, params);
  },

  /**
   * 移除成员
   */
  removeMember(data: { group_id: string; user_id: string }) {
    return post<void>(GROUPS.MEMBERS.REMOVE_MEMBER, data);
  },

  // ==================== 申请管理 ====================

  /**
   * 获取入群申请列表
   */
  getApplications(params: {
    group_id: string;
    status?: ApplicationStatus | 'all';
    pagination?: PaginationParams;
  }) {
    return paginate<GroupApplicationItem>(GROUPS.APPLICATIONS.LIST, params);
  },

  /**
   * 批准入群申请
   */
  approveApplication(data: { application_id: string }) {
    return post<void>(GROUPS.APPLICATIONS.APPROVE, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 拒绝入群申请
   */
  rejectApplication(data: { application_id: string; reason?: string }) {
    return post<void>(GROUPS.APPLICATIONS.REJECT, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  // ==================== 群公告 ====================

  /**
   * 获取群公告列表（非分页，返回所有公告）
   */
  getAnnouncements(params: { group_id: string }) {
    return post<{ items: GroupAnnouncementItem[] }>(GROUPS.ANNOUNCEMENTS.LIST, params);
  },

  /**
   * 发布群公告
   */
  createAnnouncement(data: {
    group_id: string;
    title: string;
    content: string;
    is_pinned?: boolean;
  }) {
    return post<GroupAnnouncementItem>(GROUPS.ANNOUNCEMENTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 更新群公告
   * spec: 需要 group_id + announcement_id 确定目标
   */
  updateAnnouncement(data: {
    group_id: string;
    announcement_id: string;
    title?: string;
    content?: string;
    is_pinned?: boolean;
  }) {
    return post<void>(GROUPS.ANNOUNCEMENTS.UPDATE, data);
  },

  /**
   * 删除群公告
   * spec: 需要 group_id + announcement_id 确定目标
   */
  deleteAnnouncement(data: { group_id: string; announcement_id: string }) {
    return post<void>(GROUPS.ANNOUNCEMENTS.DELETE, data);
  },

  // ==================== 当前用户 ====================

  /**
   * 获取我加入的群组列表
   */
  getMyGroups(params?: { pagination?: PaginationParams }) {
    return paginate<GroupItem>(GROUPS.ME.LIST, params);
  },

  /**
   * 获取我的入群申请列表
   */
  getMyApplications(params?: {
    status?: ApplicationStatus | 'all';
    pagination?: PaginationParams;
  }) {
    return paginate<GroupApplicationItem>(GROUPS.ME.APPLICATIONS_LIST, params);
  },

  /**
   * 取消入群申请
   */
  cancelApplication(data: { application_id: string }) {
    return post<void>(GROUPS.ME.CANCEL_APPLICATION, data);
  },

  /**
   * 获取收到的群组邀请
   */
  getInvites(params?: { status?: ApplicationStatus | 'all'; pagination?: PaginationParams }) {
    return paginate<GroupInviteItem>(GROUPS.ME.INVITES_LIST, params);
  },

  /**
   * 接受群组邀请
   */
  acceptInvite(data: { invite_id: string }) {
    return post<void>(GROUPS.ME.ACCEPT_INVITE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 拒绝群组邀请
   */
  rejectInvite(data: { invite_id: string }) {
    return post<void>(GROUPS.ME.REJECT_INVITE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 退出群组（SHOULD 幂等）
   */
  leave(data: { group_id: string }) {
    return post<void>(GROUPS.ME.LEAVE, data, { idempotencyKey: generateIdempotencyKey() });
  }
};

export const groupMessagesService = {
  /**
   * 发送群消息
   * spec: MUST 幂等，自动生成 Idempotency-Key
   */
  send(data: {
    group_id: string;
    type: SendMessageType;
    content?: string;
    file_id?: string;
    share_code?: string;
    reply_to_message_id?: string;
    mentions?: string[];
  }) {
    return post<MessageSent>(GROUP_MESSAGES.SEND, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 获取消息历史
   */
  getHistory(params: { group_id: string; pagination?: PaginationParams }) {
    return paginate<GroupMessageItem>(GROUP_MESSAGES.GET_HISTORY, params);
  },

  /**
   * 编辑消息
   */
  update(data: { message_id: string; content: string }) {
    return post<void>(GROUP_MESSAGES.UPDATE, data);
  },

  /**
   * 撤回消息
   */
  recall(data: { message_id: string }) {
    return post<void>(GROUP_MESSAGES.RECALL, data);
  },

  /**
   * 标记已读
   */
  markAsRead(data: { group_id: string; up_to_message_id?: string }) {
    return post<void>(GROUP_MESSAGES.ME.MARK_AS_READ, data);
  },

  /**
   * 标记全部群聊已读
   */
  markAllRead() {
    return post<void>(GROUP_MESSAGES.ME.MARK_ALL_READ, {});
  }
};
