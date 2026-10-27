// 消息页分栏判定与深链落地的契约测试（R13：窄平板 ?peer=/?group= 深链修复）。
import { describe, it, expect } from 'vitest';
import { canUseSplitPane, resolvePushChatHref } from '../deep-link';

describe('canUseSplitPane', () => {
	it('桌面启用 Master-Detail 分栏', () => {
		expect(canUseSplitPane({ platform: 'desktop', isWideTablet: false })).toBe(true);
	});

	it('宽平板（≥960）启用分栏', () => {
		expect(canUseSplitPane({ platform: 'tablet', isWideTablet: true })).toBe(true);
	});

	it('窄平板与移动保持单栏', () => {
		expect(canUseSplitPane({ platform: 'tablet', isWideTablet: false })).toBe(false);
		expect(canUseSplitPane({ platform: 'mobile', isWideTablet: false })).toBe(false);
	});
});

describe('resolvePushChatHref', () => {
	it('群聊深链优先落到群聊单栏页', () => {
		expect(resolvePushChatHref(null, 'group_1')).toBe('/messages/group/group_1');
		expect(resolvePushChatHref('user_sophia', 'group_1')).toBe('/messages/group/group_1');
	});

	it('私聊深链落到私聊单栏页', () => {
		expect(resolvePushChatHref('user_sophia', null)).toBe('/messages/user_sophia');
	});

	it('无深链参数时不跳转', () => {
		expect(resolvePushChatHref(null, null)).toBeNull();
	});
});
