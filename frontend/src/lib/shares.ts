import type { ShareListItem, ShareType } from '@/api/types';

// 我的分享（/me/shares）展示用纯逻辑：便于单测覆盖状态/文案分支。
// 契约口径（spec api_spec_share.md）：is_revoked=true 表示「当前不可用（已撤销或已过期）」，
// 服务端同口径且此时 preview=null；不要再自行比较 expires_at（时钟/时区口径易漂移）。

export const SHARE_TYPE_LABELS: Record<ShareType, string> = {
	moment: '瞬间',
	user: '用户',
	group: '群组',
	location: '位置'
};

export type ShareAvailability = 'active' | 'unavailable';

export function shareAvailability(item: Pick<ShareListItem, 'is_revoked'>): ShareAvailability {
	return item.is_revoked ? 'unavailable' : 'active';
}

export function shareAvailabilityLabel(item: Pick<ShareListItem, 'is_revoked'>): string {
	return item.is_revoked ? '已失效' : '有效';
}

/** 分享落地链接：统一由分享码拼绝对 URL（复制/打开共用，禁止各页面各拼一份）。 */
export function shareLink(share_code: string, origin: string): string {
	return `${origin.replace(/\/$/, '')}/share/${share_code}`;
}
