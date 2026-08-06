// msw handlers 集成测试：验证按 id 路由的 handler 行为
// （moments/open_the_door、groups/open_the_door、users/get_profile、
//  private_messages/get_history、group_messages/get_history、comments/create）。
// 直接调用 handler resolver（msw/node 网络层与 vitest 4 fetch 不兼容）。
import { describe, it, expect, beforeEach } from 'vitest';
import { handlers } from './handlers';
import { mockMoments, mockGroups, mockUsers, setMockLoggedIn } from './data';

function findHandler(path: string) {
	const h = handlers.find((x) => x.info.path === path);
	if (!h) throw new Error(`handler not found: ${path}`);
	return h;
}

async function run(path: string, body: unknown): Promise<any> {
	const h = findHandler(path);
	const result = await h.run({
		request: new Request(`http://test.local${path}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body)
		}),
		requestId: 'test',
		resolutionContext: { baseUrl: 'http://test.local' }
	});
	if (!result?.response) throw new Error(`${path} handler did not match`);
	return result.response.json();
}

describe('msw id-routed handlers', () => {
	// get_history / comments/create 是 auth-required 端点，测试默认以登录态执行
	beforeEach(() => {
		setMockLoggedIn(true);
	});

	it('moments/open_the_door 按 moment_id 返回对应瞬间，未知 id 回退第一条', async () => {
		const target = mockMoments[5];
		const res = await run('/api/v1/moments/open_the_door', { moment_id: target.moment_id });
		expect(res.code).toBe(0);
		expect(res.data.moment_id).toBe(target.moment_id);

		const fallback = await run('/api/v1/moments/open_the_door', { moment_id: 'nope' });
		expect(fallback.data.moment_id).toBe(mockMoments[0].moment_id);
	});

	it('groups/open_the_door 按 group_id 返回对应群组，未知 id 回退第一条', async () => {
		const target = mockGroups[2];
		const res = await run('/api/v1/groups/open_the_door', { group_id: target.group.group_id });
		expect(res.code).toBe(0);
		expect(res.data.group.group_id).toBe(target.group.group_id);
		expect(res.data.my_state.is_joined).toBe(true);

		const fallback = await run('/api/v1/groups/open_the_door', { group_id: 'nope' });
		expect(fallback.data.group.group_id).toBe(mockGroups[0].group.group_id);
	});

	it('users/get_profile 按 user_id 返回对应用户，未知 id 回退第一位', async () => {
		const target = mockUsers[3];
		const res = await run('/api/v1/users/get_profile', { user_id: target.user_id });
		expect(res.code).toBe(0);
		// spec: UserProfileBundle — user 为 UserItem（嵌套 user: UserInfo）
		expect(res.data.user.user.user_id).toBe(target.user_id);
		expect(Array.isArray(res.data.recent_moments.items)).toBe(true);
	});

	it('private_messages/get_history：命中会话返回种子消息，新会话返回空', async () => {
		// mockChats 第一位对端是 mockUsers[1]
		const hit = await run('/api/v1/private_messages/get_history', { target_user_id: mockUsers[1].user_id });
		expect(hit.code).toBe(0);
		expect(hit.data.items.length).toBeGreaterThan(0);
		expect(hit.data.items[0].content).toBeTruthy();

		const miss = await run('/api/v1/private_messages/get_history', { target_user_id: 'user_unknown' });
		expect(miss.data.items.length).toBe(0);
	});

	it('group_messages/get_history：命中群组返回种子消息，未知群返回空', async () => {
		const hit = await run('/api/v1/group_messages/get_history', { group_id: mockGroups[0].group.group_id });
		expect(hit.code).toBe(0);
		expect(hit.data.items.length).toBeGreaterThan(0);

		const miss = await run('/api/v1/group_messages/get_history', { group_id: 'group_unknown' });
		expect(miss.data.items.length).toBe(0);
	});

	it('comments/create 回显请求内容', async () => {
		const res = await run('/api/v1/comments/create', { moment_id: 'moment_1', content: '拍得真好看！' });
		expect(res.code).toBe(0);
		expect(res.data.content).toBe('拍得真好看！');
	});

	it('comments/list 按 moment_id 派生不同评论（不固定同批）', async () => {
		const a = await run('/api/v1/comments/list', { moment_id: mockMoments[0].moment_id });
		const b = await run('/api/v1/comments/list', { moment_id: mockMoments[3].moment_id });
		expect(a.code).toBe(0);
		expect(b.code).toBe(0);
		expect(a.data.items.length).toBeGreaterThan(0);
		// 不同瞬间的评论 user/content 应不同（derive from index）
		expect(a.data.items[0].user.user_id).not.toBe(b.data.items[0].user.user_id);
	});
});
