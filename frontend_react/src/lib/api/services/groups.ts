/**
 * 群组服务
 */

import { get, post, put, del, generateIdempotencyKey } from '../client';
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
      referencePoint ? { reference_point: referencePoint } : undefined,
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
    return put<void>(ENDPOINTS.GROUP_CHATS.UPDATE(groupId), data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async delete(groupId: string): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.GROUP_CHATS.DELETE(groupId));
  },

  async updateMySettings(
    groupId: string,
    settings: { is_muted?: boolean; is_pinned?: boolean }
  ): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.GROUP_CHATS.MY_SETTINGS(groupId), settings, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 成员管理
  // ============================================================================

  async getMembers(
    groupId: string,
    params?: { role?: 'owner' | 'admin' | 'member'; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return get<PaginatedData<GroupMemberItem>>(ENDPOINTS.GROUP_CHATS.MEMBERS(groupId), params);
  },

  async setMemberRole(groupId: string, userId: number, role: 'admin' | 'member'): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.GROUP_CHATS.MEMBER_ROLE(groupId, userId), { role }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async muteMember(groupId: string, userId: number, durationMinutes: number): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.GROUP_CHATS.MEMBER_MUTE(groupId, userId),
      { duration_minutes: durationMinutes },
      { idempotencyKey: generateIdempotencyKey() }
    );
  },

  async unmuteMember(groupId: string, userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER_UNMUTE(groupId, userId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async banMember(groupId: string, userId: number, reason?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER_BAN(groupId, userId), { reason }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async unbanMember(groupId: string, userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.MEMBER_UNBAN(groupId, userId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async getBannedMembers(
    groupId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<BannedMemberItem>>> {
    return get<PaginatedData<BannedMemberItem>>(ENDPOINTS.GROUP_CHATS.BANNED_MEMBERS(groupId), params);
  },

  async removeMember(groupId: string, userId: number): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.GROUP_CHATS.MEMBER(groupId, userId));
  },

  async transferOwnership(groupId: string, newOwnerId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.TRANSFER(groupId), { new_owner_id: newOwnerId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 加入/退出
  // ============================================================================

  async join(groupId: string, password?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.JOIN(groupId), password ? { password } : undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async leave(groupId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.LEAVE(groupId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async apply(groupId: string, message?: string): Promise<ApiResponse<{ application_id: string }>> {
    return post<{ application_id: string }>(ENDPOINTS.GROUP_CHATS.APPLY(groupId), { message }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async invite(groupId: string, userId: number, message?: string): Promise<ApiResponse<{ invite_id: string }>> {
    return post<{ invite_id: string }>(
      ENDPOINTS.GROUP_CHATS.INVITE(groupId),
      { user_id: userId, message },
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
    return get<PaginatedData<GroupApplicationItem>>(ENDPOINTS.GROUP_CHATS.APPLICATIONS(groupId), params);
  },

  async approveApplication(applicationId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.APPROVE_APPLICATION(applicationId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async rejectApplication(applicationId: string, reason?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.REJECT_APPLICATION(applicationId), { reason }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 邀请管理
  // ============================================================================

  async getMyInvites(
    params?: { status?: 'pending' | 'accepted' | 'rejected' | 'all'; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return get<PaginatedData<GroupInviteItem>>(ENDPOINTS.GROUP_CHATS.MY_INVITES, params);
  },

  async acceptInvite(inviteId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.ACCEPT_INVITE(inviteId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async rejectInvite(inviteId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_CHATS.REJECT_INVITE(inviteId), undefined, {
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
    return get<PaginatedData<GroupAnnouncementItem>>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENTS(groupId), params);
  },

  async createAnnouncement(
    groupId: string,
    data: { title: string; content: string; is_pinned?: boolean }
  ): Promise<ApiResponse<GroupAnnouncementItem>> {
    return post<GroupAnnouncementItem>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENTS(groupId), data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async updateAnnouncement(
    announcementId: string,
    data: { title?: string; content?: string; is_pinned?: boolean }
  ): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENT(announcementId), data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async deleteAnnouncement(announcementId: string): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.GROUP_CHATS.ANNOUNCEMENT(announcementId));
  },

  // ============================================================================
  // 我的群组
  // ============================================================================

  async getMyGroups(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<JoinedGroupItem>>> {
    return get<PaginatedData<JoinedGroupItem>>(ENDPOINTS.USER.ME_GROUPS, params);
  },
};
