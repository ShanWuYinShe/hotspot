// 平台管理页面的中文展示标签（纯前端展示层映射，不改 API 契约 / 数据结构）。
// 审计动作与举报原因在接口里是英文枚举键，后台面向中文运营者，统一在此翻译；
// 未知键一律回退原键（诚实降级，避免新枚举上线后页面出现空白）。

import type { AuditAction, AuditTargetType, ReportReason, ReportStatus, ReportTargetType } from '@/api/types';

/** 审计动作 → 中文（覆盖 spec api_spec_admin.md 全部 AuditAction） */
export const AUDIT_ACTION_LABELS: Record<AuditAction, string> = {
	'user.ban': '封禁用户',
	'user.unban': '解封用户',
	'group.dismiss': '解散群组',
	'group.member_mute': '禁言成员',
	'group.member_ban': '封禁成员',
	'group.member_remove': '移除成员',
	'announcement.create': '发布公告',
	'announcement.update': '更新公告',
	'announcement.delete': '删除公告',
	'role.create': '创建角色',
	'role.update': '更新角色',
	'role.delete': '删除角色',
	'role.set_permissions': '设置角色权限',
	'admin.award': '授予管理员',
	'admin.revoke': '撤销管理员',
	'report.judge': '裁决举报',
	'tag.set_ban': '标签屏蔽',
	'config.update': '更新配置'
};

/** 审计目标类型 → 中文 */
export const AUDIT_TARGET_LABELS: Record<AuditTargetType, string> = {
	user: '用户',
	group: '群组',
	announcement: '公告',
	role: '角色',
	admin: '管理员',
	report: '举报',
	tag: '标签',
	config: '配置'
};

/** 举报对象类型 → 中文 */
export const REPORT_TARGET_LABELS: Record<ReportTargetType, string> = {
	user: '用户',
	moment: '瞬间',
	comment: '评论',
	message: '消息',
	group: '群组'
};

/** 举报原因 → 中文 */
export const REPORT_REASON_LABELS: Record<ReportReason, string> = {
	spam: '垃圾信息',
	harassment: '骚扰',
	violence: '暴力',
	hate_speech: '仇恨言论',
	sexual_content: '色情内容',
	misinformation: '不实信息',
	impersonation: '冒充他人',
	self_harm: '自伤',
	other: '其他'
};

/** 举报状态 → 中文 */
export const REPORT_STATUS_LABELS: Record<ReportStatus, string> = {
	pending: '待处理',
	resolved: '已确认',
	dismissed: '已驳回'
};

/** 未知枚举键回退原键 */
export function labelOf<T extends string>(map: Record<T, string>, key: T | string): string {
	return (map as Record<string, string>)[key] ?? key;
}
