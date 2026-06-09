import { post, paginate, generateIdempotencyKey } from "../client";
import {
  ADMIN_RBAC,
  ADMIN_USERS,
  ADMIN_GROUPS,
  ADMIN_ANNOUNCEMENTS,
  ADMIN_AUDIT,
  ADMIN_REPORTS,
  ADMIN_TAGS,
  ADMIN_APP_CONFIG,
} from "../endpoints";
import type {
  AdminPermissionItem,
  AdminPermissions,
  AdminRoleItem,
  AdminOperatorItem,
  AdminUserItem,
  AdminGroupItem,
  AdminReportItem,
  AdminAnnouncementItem,
  AdminTagItem,
  AdminTagSortBy,
  AuditLogItem,
  AuditAction,
  AuditTargetType,
  AnnouncementCreated,
  AnnouncementSeverity,
  ReportHandleAction,
  ReportStatus,
  ReportTargetType,
  UserAccountStatus,
  UserType,
  AppConfig,
  UploadConfig,
  RateLimitConfig,
  ResponseDefaults,
  GroupSettings,
  PaginationParams,
  Location,
} from "@/lib/types/api";

export const adminRbacService = {
  getMyPermissions() {
    return post<AdminPermissions>(ADMIN_RBAC.GET_ME);
  },
  getPermissions() {
    return post<{ items: AdminPermissionItem[] }>(ADMIN_RBAC.LIST_PERMISSIONS);
  },
  getRoles() {
    return post<{ items: AdminRoleItem[] }>(ADMIN_RBAC.ROLES.LIST);
  },
  createRole(data: { name: string; description?: string }) {
    return post<AdminRoleItem>(ADMIN_RBAC.ROLES.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
  updateRole(data: {
    role_id: string;
    name?: string;
    description?: string;
    clear_description?: boolean;
  }) {
    return post<void>(ADMIN_RBAC.ROLES.UPDATE, data);
  },
  deleteRole(data: { role_id: string }) {
    return post<void>(ADMIN_RBAC.ROLES.DELETE, data);
  },
  setRolePermissions(data: { role_id: string; permission_keys: string[] }) {
    return post<void>(ADMIN_RBAC.ROLES.SET_PERMISSIONS, data);
  },
  getAdmins(params?: { pagination?: PaginationParams }) {
    return paginate<AdminOperatorItem>(ADMIN_RBAC.ADMINS.LIST, params);
  },
  grantAdmin(data: {
    user_id: string;
    grant_all_permissions?: boolean;
    role_ids: string[];
  }) {
    return post<void>(ADMIN_RBAC.ADMINS.GRANT, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
  revokeAdmin(data: { user_id: string }) {
    return post<void>(ADMIN_RBAC.ADMINS.REVOKE, data);
  },
};

export const adminUsersService = {
  search(params: {
    user_id?: string;
    nickname?: string;
    login_id?: string;
    status?: UserAccountStatus;
    user_type?: UserType;
    pagination?: PaginationParams;
  }) {
    return paginate<AdminUserItem>(ADMIN_USERS.SEARCH, params);
  },
  getDetail(data: { user_id: string }) {
    return post<AdminUserItem>(ADMIN_USERS.OPEN_THE_DOOR, data);
  },
  ban(data: { user_id: string; reason?: string; duration_days?: number }) {
    return post<void>(ADMIN_USERS.BAN, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
  unban(data: { user_id: string; reason?: string }) {
    return post<void>(ADMIN_USERS.UNBAN, data);
  },
};

export const adminGroupsService = {
  search(params: {
    group_id?: string;
    name?: string;
    owner_id?: string;
    pagination?: PaginationParams;
  }) {
    return paginate<AdminGroupItem>(ADMIN_GROUPS.SEARCH, params);
  },
  getDetail(data: { group_id: string }) {
    return post<AdminGroupItem>(ADMIN_GROUPS.OPEN_THE_DOOR, data);
  },
  dismiss(data: { group_id: string; reason?: string }) {
    return post<void>(ADMIN_GROUPS.DISMISS, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
};

export const adminAnnouncementsService = {
  search(params?: {
    pagination?: PaginationParams;
    severity?: AnnouncementSeverity;
  }) {
    return paginate<AdminAnnouncementItem>(ADMIN_ANNOUNCEMENTS.SEARCH, params);
  },
  create(data: {
    title: string;
    content: string;
    severity: AnnouncementSeverity;
    action_url?: string;
    send_push?: boolean;
  }) {
    return post<AnnouncementCreated>(ADMIN_ANNOUNCEMENTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
  update(data: {
    announcement_id: string;
    title?: string;
    content?: string;
    severity?: AnnouncementSeverity;
    action_url?: string;
  }) {
    return post<void>(ADMIN_ANNOUNCEMENTS.UPDATE, data);
  },
  delete(data: { announcement_id: string }) {
    return post<void>(ADMIN_ANNOUNCEMENTS.DELETE, data);
  },
};

export const adminAuditService = {
  getList(params?: {
    action?: AuditAction;
    operator_id?: string;
    target_type?: AuditTargetType;
    target_id?: string;
    start_time?: string;
    end_time?: string;
    pagination?: PaginationParams;
  }) {
    return paginate<AuditLogItem>(ADMIN_AUDIT.LIST, params);
  },
  getDetail(data: { log_id: string }) {
    return post<AuditLogItem>(ADMIN_AUDIT.OPEN_THE_DOOR, data);
  },
};

export const adminReportsService = {
  search(params?: {
    status?: ReportStatus;
    target_type?: ReportTargetType;
    pagination?: PaginationParams;
  }) {
    return paginate<AdminReportItem>(ADMIN_REPORTS.SEARCH, params);
  },
  handle(data: {
    report_id: string;
    action: ReportHandleAction;
    note?: string;
  }) {
    return post<void>(ADMIN_REPORTS.HANDLE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
};

export const adminTagsService = {
  search(params?: {
    keyword?: string;
    sort_by?: AdminTagSortBy;
    pagination?: PaginationParams;
  }) {
    return paginate<AdminTagItem>(ADMIN_TAGS.SEARCH, params);
  },
  delete(data: { name: string }) {
    return post<void>(ADMIN_TAGS.DELETE, data);
  },
};

export const adminAppConfigService = {
  get() {
    return post<AppConfig>(ADMIN_APP_CONFIG.GET);
  },
  updateUpload(data: { upload: Partial<UploadConfig> }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_UPLOAD, data);
  },
  updateDefaults(data: {
    default_location?: Partial<Location>;
    default_group_avatar?: string;
    default_user_avatar?: string;
    default_group_settings?: Partial<GroupSettings>;
  }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_DEFAULTS, data);
  },
  updateRateLimits(data: { rate_limits: Partial<RateLimitConfig> }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_RATE_LIMITS, data);
  },
  updateResponseDefaults(data: {
    response_defaults: Partial<ResponseDefaults>;
  }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_RESPONSE_DEFAULTS, data);
  },
  updateFeatures(data: {
    enable_public_registration?: boolean;
    enable_guest_login?: boolean;
    enable_moments?: boolean;
    enable_nearby_people?: boolean;
  }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_FEATURES, data);
  },
  updateLimits(data: {
    limits: Partial<{
      max_group_members: number;
      max_moment_tags: number;
      max_moment_images: number;
      moment_description_max_length: number;
      comment_max_length: number;
      search_radius_max: number;
    }>;
  }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_LIMITS, data);
  },
  updateContact(data: {
    support_email?: string;
    privacy_policy_url?: string;
    terms_of_service_url?: string;
    official_website?: string;
  }) {
    return post<void>(ADMIN_APP_CONFIG.UPDATE_CONTACT, data);
  },
};
