/**
 * 管理员服务
 */

import { post, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  AdminMeData,
  AdminPermissionItem,
  AdminRoleItem,
  AdminUserItem,
  AdminGroupItem,
  ReportItem,
  AnnouncementItem,
  AuditLogItem,
} from '@/types/api';

export const adminService = {
  // ============================================================================
  // RBAC
  // ============================================================================

  async getMe(): Promise<ApiResponse<AdminMeData>> {
    return post<AdminMeData>(ENDPOINTS.ADMIN.ME, undefined, { requireAuth: true });
  },

  async getPermissions(): Promise<ApiResponse<{ items: AdminPermissionItem[] }>> {
    return post<{ items: AdminPermissionItem[] }>(ENDPOINTS.ADMIN.PERMISSIONS, undefined, { requireAuth: true });
  },

  async getRoles(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<AdminRoleItem>>> {
    return post<PaginatedData<AdminRoleItem>>(ENDPOINTS.ADMIN.ROLES, params, { requireAuth: true });
  },

  async createRole(data: { name: string; description?: string; permissions: string[] }): Promise<ApiResponse<AdminRoleItem>> {
    return post<AdminRoleItem>(ENDPOINTS.ADMIN.ROLES, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async updateRole(roleId: string, data: { name?: string; description?: string; permissions?: string[] }): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.ROLE(roleId), data, {
      requireAuth: true,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async deleteRole(roleId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.ROLE(roleId), { action: 'delete' }, {
      requireAuth: true,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 用户管理
  // ============================================================================

  async searchUsers(params: {
    keyword?: string;
    user_type?: string;
    is_banned?: boolean;
    limit?: number;
    cursor?: string;
  }): Promise<ApiResponse<PaginatedData<AdminUserItem>>> {
    return post<PaginatedData<AdminUserItem>>(ENDPOINTS.ADMIN.USERS_SEARCH, params, { requireAuth: true });
  },

  async getUserDetail(userId: number): Promise<ApiResponse<AdminUserItem>> {
    return post<AdminUserItem>(ENDPOINTS.ADMIN.USER_DETAIL(userId), undefined, { requireAuth: true });
  },

  async banUser(userId: number, data: { reason?: string; duration_days?: number }): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.USER_BAN(userId), data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async unbanUser(userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.USER_UNBAN(userId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 群组管理
  // ============================================================================

  async searchGroups(params: {
    keyword?: string;
    is_active?: boolean;
    limit?: number;
    cursor?: string;
  }): Promise<ApiResponse<PaginatedData<AdminGroupItem>>> {
    return post<PaginatedData<AdminGroupItem>>(ENDPOINTS.ADMIN.GROUPS_SEARCH, params, { requireAuth: true });
  },

  async getGroupDetail(groupId: string): Promise<ApiResponse<AdminGroupItem>> {
    return post<AdminGroupItem>(ENDPOINTS.ADMIN.GROUP_DETAIL(groupId), undefined, { requireAuth: true });
  },

  async dismissGroup(groupId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.GROUP_DISMISS(groupId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 举报管理
  // ============================================================================

  async getReports(params?: {
    status?: string;
    target_type?: string;
    limit?: number;
    cursor?: string;
  }): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return post<PaginatedData<ReportItem>>(ENDPOINTS.ADMIN.REPORTS, params, { requireAuth: true });
  },

  async handleReport(reportId: string, data: {
    action: 'resolve' | 'dismiss' | 'escalate';
    note?: string;
  }): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.REPORT_HANDLE(reportId), data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 公告管理
  // ============================================================================

  async getAnnouncements(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<AnnouncementItem>>> {
    return post<PaginatedData<AnnouncementItem>>(ENDPOINTS.ADMIN.ANNOUNCEMENTS, params, { requireAuth: false });
  },

  async createAnnouncement(data: {
    title: string;
    content: string;
    severity: 'info' | 'warning' | 'error';
    is_active?: boolean;
    starts_at?: string;
    ends_at?: string;
  }): Promise<ApiResponse<AnnouncementItem>> {
    return post<AnnouncementItem>(ENDPOINTS.ADMIN.ANNOUNCEMENTS, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async updateAnnouncement(announcementId: string, data: {
    title?: string;
    content?: string;
    severity?: 'info' | 'warning' | 'error';
    is_active?: boolean;
    starts_at?: string;
    ends_at?: string;
  }): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.ANNOUNCEMENT(announcementId), data, {
      requireAuth: true,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  async deleteAnnouncement(announcementId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.ADMIN.ANNOUNCEMENT(announcementId), { action: 'delete' }, {
      requireAuth: true,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 审计日志
  // ============================================================================

  async getAuditLogs(params?: {
    admin_user_id?: number;
    action?: string;
    start_time?: string;
    end_time?: string;
    limit?: number;
    cursor?: string;
  }): Promise<ApiResponse<PaginatedData<AuditLogItem>>> {
    return post<PaginatedData<AuditLogItem>>(ENDPOINTS.ADMIN.AUDIT_LOGS, params, { requireAuth: true });
  },

  async exportAuditLogs(params: {
    start_time: string;
    end_time: string;
    admin_user_id?: number;
    action?: string;
  }): Promise<ApiResponse<{ download_url: string }>> {
    return post<{ download_url: string }>(ENDPOINTS.ADMIN.AUDIT_EXPORT, params);
  },
};
