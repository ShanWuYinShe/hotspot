// 群详情 tab URL 契约（W20）：默认档 / 合法值 / 非法回落 / 成员子档归属。
import { describe, it, expect } from 'vitest';
import { groupTabQuery, parseGroupTab, parseMemberTab } from './group-tabs';

describe('group-tabs', () => {
	it('合法 tab 原样返回，默认 overview', () => {
		expect(parseGroupTab('members')).toBe('members');
		expect(parseGroupTab('announcements')).toBe('announcements');
		expect(parseGroupTab(null)).toBe('overview');
		expect(parseGroupTab('bogus')).toBe('overview');
	});

	it('成员子档合法值解析，非法回落 members', () => {
		expect(parseMemberTab('applications')).toBe('applications');
		expect(parseMemberTab('invites')).toBe('invites');
		expect(parseMemberTab('banned')).toBe('banned');
		expect(parseMemberTab('')).toBe('members');
		expect(parseMemberTab('nope')).toBe('members');
	});

	it('overview 不写 tab；成员子档只在 members 档下保留', () => {
		expect(groupTabQuery('overview', 'members')).toEqual({ tab: null, memberTab: null });
		expect(groupTabQuery('members', 'members')).toEqual({ tab: 'members', memberTab: null });
		expect(groupTabQuery('members', 'applications')).toEqual({ tab: 'members', memberTab: 'applications' });
		// 切到公告档：孤儿 memberTab 必须清掉
		expect(groupTabQuery('announcements', 'applications')).toEqual({ tab: 'announcements', memberTab: null });
	});
});
