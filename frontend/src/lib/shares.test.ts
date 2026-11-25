import { describe, expect, it } from 'vitest';
import { shareAvailability, shareAvailabilityLabel, shareLink, SHARE_TYPE_LABELS } from './shares';

describe('shares helpers', () => {
	it('is_revoked 与过期同口径：true=已失效', () => {
		expect(shareAvailability({ is_revoked: false })).toBe('active');
		expect(shareAvailability({ is_revoked: true })).toBe('unavailable');
		expect(shareAvailabilityLabel({ is_revoked: false })).toBe('有效');
		expect(shareAvailabilityLabel({ is_revoked: true })).toBe('已失效');
	});

	it('份额码链接拼接不重复斜杠', () => {
		expect(shareLink('ABC234567890', 'https://chaoset.com')).toBe('https://chaoset.com/share/ABC234567890');
		expect(shareLink('ABC234567890', 'https://chaoset.com/')).toBe('https://chaoset.com/share/ABC234567890');
		expect(shareLink('ABC234567890', 'http://localhost:8888')).toBe('http://localhost:8888/share/ABC234567890');
	});

	it('类型中文文案覆盖四种类型', () => {
		expect(SHARE_TYPE_LABELS).toEqual({ moment: '瞬间', user: '用户', group: '群组', location: '位置' });
	});
});
