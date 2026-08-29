// ==================== Admin Types ====================
import { AnnounceSeverity, GroupJoinMethod, PaginationParams, RelativeLocation, ReportAction, ReportStatus, ReportTargetType, UserAccountStatus, UserPresenceStatus, UserType } from './common';
import { GroupInfo, UserInfo, UserStats } from './entities';


export interface AdminUserItem {
  user: UserInfo;
  status: UserAccountStatus;
  created_at: string;
  last_active_at: string | null;
  login_id: string;
  presence_status: UserPresenceStatus | null;
}

export interface AdminUserDetail {
  user: UserInfo;
  status: UserAccountStatus;
  created_at: string;
  last_active_at: string | null;
  login_id: string;
  email: string | null;
  phone: string | null;
  bio: string | null;
  tags: string[];
  location: RelativeLocation | null;
  presence_status: UserPresenceStatus | null;
  stats: UserStats;
  ban_history?: AdminBanRecord[];
}

export interface AdminBanRecord {
  banned_at: string;
  banned_by: UserInfo;
  reason?: string;
  duration_days?: number;
  unbanned_at?: string;
  unbanned_by?: UserInfo;
}

export interface AdminGroupItem {
  group: GroupInfo;
  owner: UserInfo;
  created_at: string;
}

export interface AdminGroupDetailSettings {
  join_method: GroupJoinMethod;
  is_searchable: boolean;
  has_password: boolean;
}

export interface AdminGroupDetailStats {
  messages_count: number;
  active_members_today: number;
}

export interface AdminGroupDetail {
  group: GroupInfo;
  description: string | null;
  tags: string[];
  max_members: number;
  owner: UserInfo;
  created_at: string;
  location: RelativeLocation | null;
  settings: AdminGroupDetailSettings;
  admins?: UserInfo[];
  stats: AdminGroupDetailStats;
  dismissal_history?: AdminDismissalRecord[];
}

export interface AdminDismissalRecord {
  dismissed_at: string;
  dismissed_by: UserInfo;
  reason?: string;
}

// Admin RBAC Types
export interface AdminPermissions {
  permissions: string[];
  roles: AdminRoleBasic[];
}

export interface AdminRoleBasic {
  role_id: string;
  name: string;
}

export interface AdminPermissionItem {
  key: string;
  description: string;
}

export interface AdminRoleItem {
  role_id: string;
  name: string;
  description?: string;
  permissions: string[];
}

export interface AdminAdminItem {
  user: UserInfo;
  permissions: string[];
  roles: AdminRoleBasic[];
}

export interface AdminRbacRoleCreateRequest {
  name: string;
  description?: string;
}

export interface AdminRbacRoleUpdateRequest {
  role_id: string;
  name?: string;
  description?: string;
  clear_description?: boolean;
}

export interface AdminRbacRoleSetPermissionsRequest {
  role_id: string;
  permission_keys: string[];
}

export interface AdminRbacAwardAdminRequest {
  user_id: string;
  award_all_permissions?: boolean;
  role_ids: string[];
}

export interface AdminRbacRevokeAdminRequest {
  user_id: string;
}

// Admin Audit Types
export type AuditAction =
  | 'user.ban'
  | 'user.unban'
  | 'group.dismiss'
  | 'announcement.create'
  | 'announcement.delete'
  | 'role.create'
  | 'role.update'
  | 'role.delete'
  | 'permission.grant'
  | 'permission.revoke'
  | 'config.update';

export type AuditTargetType = 'user' | 'group' | 'announcement' | 'role' | 'config';

export interface AuditLogItem {
  log_id: string;
  action: AuditAction;
  operator: UserInfo;
  target_type: AuditTargetType;
  target_id: string;
  target_name: string | null;
  details: AuditDetails | null;
  ip_location: string | null;
  ip_address: string | null;
  device_summary: string | null;
  created_at: string;
}

export interface AuditDetailsUserBan {
  type: 'user.ban';
  reason?: string;
  duration_days?: number;
}

export interface AuditDetailsUserUnban {
  type: 'user.unban';
  reason?: string;
}

export interface AuditDetailsGroupDismiss {
  type: 'group.dismiss';
  reason?: string;
}

export interface AuditDetailsAnnouncementCreate {
  type: 'announcement.create';
  title: string;
}

export interface AuditDetailsAnnouncementDelete {
  type: 'announcement.delete';
  title: string;
}

export interface AuditDetailsRoleCreate {
  type: 'role.create';
  role_name: string;
  permissions: string[];
}

export interface AuditDetailsRoleUpdate {
  type: 'role.update';
  role_name: string;
  added_permissions?: string[];
  removed_permissions?: string[];
}

export interface AuditDetailsRoleDelete {
  type: 'role.delete';
  role_name: string;
}

export interface AuditDetailsPermissionGrant {
  type: 'permission.grant';
  permission: string;
}

export interface AuditDetailsPermissionRevoke {
  type: 'permission.revoke';
  permission: string;
}

export interface AuditDetailsConfigUpdate {
  type: 'config.update';
  config_key: string;
}

export type AuditDetails =
  | AuditDetailsUserBan
  | AuditDetailsUserUnban
  | AuditDetailsGroupDismiss
  | AuditDetailsAnnouncementCreate
  | AuditDetailsAnnouncementDelete
  | AuditDetailsRoleCreate
  | AuditDetailsRoleUpdate
  | AuditDetailsRoleDelete
  | AuditDetailsPermissionGrant
  | AuditDetailsPermissionRevoke
  | AuditDetailsConfigUpdate;

export interface AuditLogDetail extends AuditLogItem {
  before_state?: StateSnapshot;
  after_state?: StateSnapshot;
}

export interface StateSnapshotUser {
  type: 'user';
  nickname?: string;
  status?: string;
  roles?: string[];
}

export interface StateSnapshotGroup {
  type: 'group';
  name?: string;
  members_count?: number;
}

export interface StateSnapshotRole {
  type: 'role';
  name?: string;
  permissions?: string[];
}

export interface StateSnapshotConfig {
  type: 'config';
  value?: string;
}

export type StateSnapshot =
  | StateSnapshotUser
  | StateSnapshotGroup
  | StateSnapshotRole
  | StateSnapshotConfig;

export interface AuditLogListRequest {
  action?: AuditAction;
  operator_id?: string;
  target_type?: AuditTargetType;
  target_id?: string;
  start_time?: string;
  end_time?: string;
  pagination?: PaginationParams;
}

export interface AuditLogDetailRequest {
  log_id: string;
}

export interface AuditLogExportRequest {
  action?: AuditAction;
  operator_id?: string;
  target_type?: AuditTargetType;
  start_time?: string;
  end_time?: string;
  format?: 'csv' | 'json';
}

export interface AuditLogExportResult {
  export_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  download_url?: string;
  expires_at?: string;
}

export interface AuditLogExportStatusRequest {
  export_id: string;
}

// Admin Announcement Types
export interface CreateAnnouncementRequest {
  title: string;
  content: string;
  severity: AnnounceSeverity;
  action_url?: string;
  send_push?: boolean;
}

export interface AnnouncementCreated {
  announcement_id: string;
}

// Admin Report Types
export interface AdminReportSearchRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}

export interface JudgeAdminReportRequest {
  report_id: string;
  action: ReportAction;
  note?: string;
  ban_duration_days?: number;
}

// Admin User Types
export interface AdminUserSearchRequest {
  user_id?: string;
  nickname?: string;
  login_id?: string;
  status?: UserAccountStatus;
  user_type?: UserType;
  pagination?: PaginationParams;
}

export interface AdminUserDetailRequest {
  user_id: string;
}

export interface AdminBanUserRequest {
  user_id: string;
  reason?: string;
  duration_days?: number;
}

export interface AdminUnbanUserRequest {
  user_id: string;
  reason?: string;
}

export interface AdminGroupSearchRequest {
  group_id?: string;
  name?: string;
  owner_id?: string;
  pagination?: PaginationParams;
}

export interface AdminGroupDetailRequest {
  group_id: string;
}

export interface AdminDismissGroupRequest {
  group_id: string;
  reason?: string;
}
