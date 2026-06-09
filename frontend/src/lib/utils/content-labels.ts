import type { ReportReason, ReportStatus, ReportTargetType, ShareType } from '$lib/types/api';

export type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

export const SHARE_TYPE_LABEL: Record<ShareType, string> = {
  user: '用户',
  group: '群组',
  moment: '动态',
  location: '位置'
};

export const SHARE_TYPE_BADGE_CLASS: Record<ShareType, string> = {
  user: 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-800',
  group:
    'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-300 dark:border-emerald-800',
  moment:
    'bg-orange-50 text-orange-700 border-orange-200 dark:bg-orange-900/30 dark:text-orange-300 dark:border-orange-800',
  location:
    'bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-900/30 dark:text-violet-300 dark:border-violet-800'
};

export const REPORT_REASON_LABEL: Record<ReportReason, string> = {
  spam: '垃圾内容',
  harassment: '骚扰',
  violence: '暴力',
  hate_speech: '仇恨言论',
  sexual_content: '色情内容',
  misinformation: '虚假信息',
  impersonation: '冒充',
  self_harm: '自残',
  other: '其他'
};

export const REPORT_TARGET_TYPE_LABEL: Record<ReportTargetType, string> = {
  user: '用户',
  moment: '瞬间',
  comment: '评论',
  message: '消息',
  group: '群组'
};

export const REPORT_STATUS_LABEL: Record<ReportStatus, string> = {
  pending: '待处理',
  reviewing: '审核中',
  resolved: '已处理',
  dismissed: '已驳回'
};

export const REPORT_STATUS_BADGE_CLASS: Record<ReportStatus, string> = {
  pending:
    'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-300',
  reviewing:
    'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  resolved:
    'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300',
  dismissed:
    'border-slate-200 bg-slate-100 text-slate-700 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-300'
};

export const REPORT_STATUS_BADGE_VARIANT: Record<ReportStatus, BadgeVariant> = {
  pending: 'secondary',
  reviewing: 'default',
  resolved: 'default',
  dismissed: 'outline'
};
