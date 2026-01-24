/**
 * 群组服务
 */

import { post, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  GroupItem,
  GroupDetailData,
  JoinedGroupItem,
  GroupMemberItem,
  GroupApplicationItem,
  GroupInviteItem,
  BannedMemberItem,
  GroupAnnouncementItem,
  CreateGroupChatRequest,
  GroupSettings,
  GeoPoint,
} from '@/types/api';

export interface SearchByKeywordParams {
  kw: string;
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface SearchByTagParams {
  tag: string;
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface SearchByLocationParams {
  reference_point: GeoPoint;
  radius: number;
  kw?: string;
  limit?: number;
  cursor?: string;
}

export const groupsService = {
  // ============================================================================
  // 搜索
  // ============================================================================

  async searchByKeyword(params: SearchByKeywordParams): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return post<PaginatedData<GroupItem>>(ENDPOINTS.GROUP_CHATS.SEARCH_BY_KEYWORD, params, {
      requireAuth: false,
    });
  },

  async searchByTag(params: SearchByTagParams): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return post<PaginatedData<GroupItem>>(ENDPOINTS.GROUP_CHATS.SEARCH_BY_TAG, params, {
      requireAuth: false,
    });
  },

  async searchByLocation(params: SearchByLocationParams): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return post<PaginatedData<GroupItem>>(ENDPOINTS.GROUP_CHATS.SEARCH_BY_LOCATION, params, {
      requireAuth: false,
    });
  },

  // ============================================================================
  // CRUD
  // ============================================================================

  async getDetail(groupId: string, referencePoint?: GeoPoint): Promise<ApiResponse<GroupDetailData>> {
    return post<GroupDetailData>(
      ENDPOINTS.GROUP_CHATS.DETAIL(groupId),
      {
        group_chat_id: groupId,
        ...(referencePoint ? { reference_point: referencePoint } : {}),
      },
      { requireAuth: false }
    );
  },

  async create(data: CreateGroupChatRequest): Promise<ApiResponse<JoinedGroupItem>> {
    return post<JoinedGroupItem>(ENDPOINTS.GROUP_CHATS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async update(
    groupId: string,
    data: {
      name?: string;
      description?: string;
      avatar_file_id?: string;
      tags?: string[];
      settings?: GroupSettings;
    }
  ): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.UPDATE(groupId), { group_chat_id: groupId, ...data }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async delete(groupId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.DELETE(groupId), { group_chat_id: groupId });
  },

  async updateMySettings(
    groupId: string,
    settings: { is_muted?: boolean; is_pinned?: boolean }
  ): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.GROUP_CHATS.MY_SETTINGS(groupId),
      { group_chat_id: groupId, ...settings },
      {
      idempotencyKey: generateIdempotencyKey(),
      }
    );
  },

  // ============================================================================
  // 成员管理
  // ============================================================================

  async getMembers(
    groupId: string,
    params?: { role?: 'owner' | 'admin' | 'member'; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return post<PaginatedData<GroupMemberItem>>(ENDPOINTS.GROUP_CHATS.MEMBERS(groupId), {
      group_chat_id: groupId,
      ...(params ?? {}),
    });
  },

  async setMemberRole(groupId: string, userId: number, role: 'admin' | 'member'): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.GROUP_CHATS.MEMBER_ROLE(groupId, userId),
      { group_chat_id: groupId, user_id: userId, role },
      {
      idempotencyKey: generateIdempotencyKey(),
      }
    );
  },

  async muteMember(groupId: string, userId: number, durationMinutes: number): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.GROUP_CHATS.MEMBER_MUTE(groupId, userId),
      { group_chat_id: groupId, user_id: userId, duration_minutes: durationMinutes },
      { idempotencyKey: generateIdempotencyKey() }
    );
  },

  async unmuteMember(groupId: string, userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER_UNMUTE(groupId, userId), {
      group_chat_id: groupId,
      user_id: userId,
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async banMember(groupId: string, userId: number, reason?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER_BAN(groupId, userId), {
      group_chat_id: groupId,
      user_id: userId,
      reason,
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async unbanMember(groupId: string, userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER_UNBAN(groupId, userId), {
      group_chat_id: groupId,
      user_id: userId,
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async getBannedMembers(
    groupId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<BannedMemberItem>>> {
    return post<PaginatedData<BannedMemberItem>>(ENDPOINTS.GROUP_CHATS.BANNED_MEMBERS(groupId), {
      group_chat_id: groupId,
      ...(params ?? {}),
    });
  },

  async removeMember(groupId: string, userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER(groupId, userId), {
      group_chat_id: groupId,
      user_id: userId,
    });
  },

  async transferOwnership(groupId: string, newOwnerId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.TRANSFER(groupId), {
      group_chat_id: groupId,
      new_owner_id: newOwnerId,
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 加入/退出
  // ============================================================================

  async join(groupId: string, password?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.JOIN(groupId), {
      group_chat_id: groupId,
      ...(password ? { password } : {}),
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async leave(groupId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.LEAVE(groupId), { group_chat_id: groupId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async apply(groupId: string, message?: string): Promise<ApiResponse<{ application_id: string }>> {
    return post<{ application_id: string }>(ENDPOINTS.GROUP_CHATS.APPLY(groupId), {
      group_chat_id: groupId,
      message,
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async invite(groupId: string, userId: number, message?: string): Promise<ApiResponse<{ invite_id: string }>> {
    return post<{ invite_id: string }>(
      ENDPOINTS.GROUP_CHATS.INVITE(groupId),
      { group_chat_id: groupId, user_id: userId, message },
      { idempotencyKey: generateIdempotencyKey() }
    );
  },

  // ============================================================================
  // 申请管理
  // ============================================================================

  async getApplications(
    groupId: string,
    params?: { status?: 'pending' | 'accepted' | 'rejected' | 'all'; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return post<PaginatedData<GroupApplicationItem>>(ENDPOINTS.GROUP_CHATS.APPLICATIONS(groupId), {
      group_chat_id: groupId,
      ...(params ?? {}),
    });
  },

  async approveApplication(applicationId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.APPROVE_APPLICATION(applicationId), { application_id: applicationId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async rejectApplication(applicationId: string, reason?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.REJECT_APPLICATION(applicationId), { application_id: applicationId, reason }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 邀请管理
  // ============================================================================

  async getMyInvites(
    params?: { status?: 'pending' | 'accepted' | 'rejected' | 'all'; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return post<PaginatedData<GroupInviteItem>>(ENDPOINTS.GROUP_CHATS.MY_INVITES, params ?? {});
  },

  async acceptInvite(inviteId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.ACCEPT_INVITE(inviteId), { invite_id: inviteId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async rejectInvite(inviteId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.REJECT_INVITE(inviteId), { invite_id: inviteId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 公告
  // ============================================================================

  async getAnnouncements(
    groupId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<GroupAnnouncementItem>>> {
    return post<PaginatedData<GroupAnnouncementItem>>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENTS(groupId), {
      group_chat_id: groupId,
      ...(params ?? {}),
    });
  },

  async createAnnouncement(
    groupId: string,
    data: { title: string; content: string; is_pinned?: boolean }
  ): Promise<ApiResponse<GroupAnnouncementItem>> {
    return post<GroupAnnouncementItem>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENTS(groupId), {
      group_chat_id: groupId,
      ...data,
    }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async updateAnnouncement(
    announcementId: string,
    data: { title?: string; content?: string; is_pinned?: boolean }
  ): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENT(announcementId), { announcement_id: announcementId, ...data }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async deleteAnnouncement(announcementId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENT(announcementId), { announcement_id: announcementId });
  },

  // ============================================================================
  // 我的群组
  // ============================================================================

  async getMyGroups(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<JoinedGroupItem>>> {
    return post<PaginatedData<JoinedGroupItem>>(ENDPOINTS.USER.ME_GROUPS, params ?? {});
  },
};
