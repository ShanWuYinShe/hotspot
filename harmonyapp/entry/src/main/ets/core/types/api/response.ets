// ==================== Response 包装类型 (ArkTS 要求命名类型) ====================
import { SessionItem } from './auth';
import { AnnounceSeverity, FollowSource, PaginationParams } from './common';
import { GroupAnnouncementItem, MomentSummary, SystemAnnouncementItem, UserInfo } from './entities';


export interface SystemAnnouncementListData {
  items: SystemAnnouncementItem[];
}

export interface SessionListData {
  items: SessionItem[];
}

export interface GroupAnnouncementListData {
  items: GroupAnnouncementItem[];
}

// ==================== 通知中心 API 类型 ====================

export type NotifyType =
  'new_follower' | 'mutual_follow' | 'moment_commented'
  | 'comment_reply' | 'mentioned' | 'group_mentioned'
  | 'reaction_added'
  | 'group_invite_received'
  | 'group_application_received' | 'group_application_accepted' | 'group_application_rejected'
  | 'report_resolved' | 'report_dismissed'
  | 'system_announcement'
  | 'nearby_new_moment';

// Spec 定义 NotifyPayload 为 discriminated union（按 type 区分），
// 但 ArkTS 当前不支持基于字面量类型的类型窄化（type narrowing），
// 无法在 if/switch 中自动推导 payload 的具体字段。
// 因此使用扁平接口替代，运行时由业务层按 type 字段手动判断。
// 若未来 ArkTS 支持 discriminated union 窄化，应迁移至联合类型以获得编译期安全。
export interface NotifyPayload {
  type: NotifyType;
  actor?: UserInfo;
  applicant?: UserInfo;
  user?: UserInfo;
  source?: FollowSource;
  source_moment_id?: string;
  moment_id?: string;
  moment_summary?: MomentSummary;
  comment_id?: string;
  comment_preview?: string;
  replied_comment_preview?: string;
  reply_id?: string;
  reply_content_preview?: string;
  target_type?: 'message' | 'moment' | 'comment';
  emoji?: string;
  group_id?: string;
  group_name?: string;
  group_avatar_url?: string;
  message_id?: string;
  content_preview?: string;
  invite_id?: string;
  message?: string;
  application_id?: string;
  announcement_id?: string;
  title?: string; // system_announcement：公告标题（spec notify.md）
  severity?: AnnounceSeverity;
  revoked_by?: UserInfo;
  rejected_by?: UserInfo;
  reason?: string;
  report_id?: string;
  target_id?: string;
  resolved_at?: string;
  dismissed_at?: string;
  note?: string;
  distance?: number;
}

export interface NotifyItem {
  notify_id: string;
  type: NotifyType;
  is_read: boolean;
  created_at: string;
  payload: NotifyPayload;
}

export interface UnreadCountByType {
  social: number;
  interaction: number;
  group: number;
  report: number;
  nearby: number;
  system: number;
}

export interface UnreadSummaryResult {
  total: number;
  by_type: UnreadCountByType;
}

export interface FetchNotifyRequest {
  type?: NotifyType;
  is_read?: boolean;
  pagination?: PaginationParams;
}

// spec api_spec_notify.md → notify/mark_read：scope 必显式（'ids' 需 notify_ids 1-50；'all' 全部已读）。
export interface NotifyMarkReadRequest {
  scope: 'ids' | 'all';
  notify_ids?: string[];
}
