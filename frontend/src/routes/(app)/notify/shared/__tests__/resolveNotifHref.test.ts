import { describe, it, expect } from 'vitest';
import { resolveNotifHref } from '../notify-data';

// resolveNotifHref：通知行点击跳转路由映射（全类型统一口径）。
describe('resolveNotifHref', () => {
	it('瞬间类通知路由到瞬间详情', () => {
		for (const type of ['nearby_new_moment', 'moment_commented', 'comment_reply', 'mentioned']) {
			expect(resolveNotifHref(type, { moment_id: 'moment_x' }, 'u_actor')).toBe('/moments/moment_x');
		}
	});

	it('reaction_added 仅 moment 目标路由到瞬间详情', () => {
		expect(resolveNotifHref('reaction_added', { target_type: 'moment', target_id: 'moment_y' }, 'u1')).toBe('/moments/moment_y');
		expect(resolveNotifHref('reaction_added', { target_type: 'comment', target_id: 'comment_x' }, 'u1')).toBeUndefined();
	});

	it('关注类通知路由到 actor 主页', () => {
		expect(resolveNotifHref('new_follower', {}, 'u_followed')).toBe('/users/u_followed');
		expect(resolveNotifHref('mutual_follow', {}, 'u_followed')).toBe('/users/u_followed');
	});

	it('群类通知路由到群详情', () => {
		for (const type of [
			'group_invite_received',
			'group_application_received',
			'group_application_accepted',
			'group_application_rejected',
			'group_mentioned'
		]) {
			expect(resolveNotifHref(type, { group_id: 'group_x' }, 'u1')).toBe('/groups/group_x');
		}
	});

	it('系统公告与举报处理通知不跳转', () => {
		expect(resolveNotifHref('system_announcement', {}, 'u1')).toBeUndefined();
		expect(resolveNotifHref('report_resolved', {}, 'u1')).toBeUndefined();
		expect(resolveNotifHref('report_dismissed', {}, 'u1')).toBeUndefined();
	});

	it('缺 payload 上下文时优雅回退 undefined', () => {
		expect(resolveNotifHref('nearby_new_moment', {}, 'u1')).toBeUndefined();
		expect(resolveNotifHref('new_follower', {}, '')).toBeUndefined();
	});
});
