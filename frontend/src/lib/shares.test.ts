import { describe, expect, it } from 'vitest';
import type { ShareListItem } from '@/api/types';
import { filterShares, shareLink, shareAvailability, shareAvailabilityLabel } from './shares';

const items: ShareListItem[] = [
	{
		share_id: 's1',
		share_code: 'abc123',
		share_type: 'moment',
		created_at: '2026-11-28T00:00:00Z',
		expires_at: null,
		is_revoked: false,
		preview: { title: '云龙湖落日', subtitle: '阿禾的瞬间', cover_url: null }
	},
	{
		share_id: 's2',
		share_code: 'def456',
		share_type: 'group',
		created_at: '2026-11-27T00:00:00Z',
		expires_at: null,
		is_revoked: true,
		preview: null
	}
];

describe('filterShares', () => {
	it('空词返回原始列表', () => {
		expect(filterShares(items, '   ')).toEqual(items);
	});

	it('匹配标题与副标题', () => {
		expect(filterShares(items, '云龙湖').map((i) => i.share_id)).toEqual(['s1']);
		expect(filterShares(items, '阿禾').map((i) => i.share_id)).toEqual(['s1']);
	});

	it('匹配分享类型文案与分享码', () => {
		expect(filterShares(items, '群组').map((i) => i.share_id)).toEqual(['s2']);
		expect(filterShares(items, 'abc123').map((i) => i.share_id)).toEqual(['s1']);
	});

	it('匹配可用状态', () => {
		expect(filterShares(items, '已失效').map((i) => i.share_id)).toEqual(['s2']);
	});

	it('无匹配返回空数组', () => {
		expect(filterShares(items, '不存在的关键词')).toEqual([]);
	});
});

describe('shareLink / shareAvailability', () => {
	it('shareLink 拼接绝对地址并去尾斜杠', () => {
		expect(shareLink('abc123', 'https://chaoset.com')).toBe('https://chaoset.com/share/abc123');
		expect(shareLink('abc123', 'https://chaoset.com/')).toBe('https://chaoset.com/share/abc123');
	});

	it('is_revoked 决定可用状态与文案（与 spec 同口径，不看 expires_at）', () => {
		const expired = { is_revoked: true } as unknown as ShareListItem;
		const active = { is_revoked: false } as unknown as ShareListItem;
		expect(shareAvailability(expired)).toBe('unavailable');
		expect(shareAvailabilityLabel(expired)).toBe('已失效');
		expect(shareAvailability(active)).toBe('active');
		expect(shareAvailabilityLabel(active)).toBe('有效');
	});

	it('空分享码也能拼出 URL（不抛错，由调用方校验）', () => {
		expect(shareLink('', 'https://chaoset.com')).toBe('https://chaoset.com/share/');
	});
});
