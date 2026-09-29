// useReactedMoments 的 user_id 契约：list_reacted 必传 user_id 且不回退当前用户，
// 传空串后端回 10000。hook 必须显式传调用方给的 id；未登录时不发请求、直接 needsAuth。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import ReactedHarness from './ReactedHarness.svelte';
import type { useReactedMoments } from '../profile-data.svelte';

vi.mock('@/api', () => ({
	listUserReactedMoments: vi.fn()
}));

import * as api from '@/api';

type R = ReturnType<typeof useReactedMoments>;

function mount(uid: () => string | null | undefined): Promise<R> {
	return new Promise((resolve) => {
		render(ReactedHarness, { props: { uid, onReady: (r: R) => resolve(r) } });
	});
}

const momentItem = {
	moment_id: 'm1',
	description: '喜欢的瞬间',
	reactions: [{ reacted_by_me: true, count: 1 }],
	comments_count: 0
};

beforeEach(() => {
	vi.clearAllMocks();
});

describe('useReactedMoments user_id', () => {
	it('有 id 时带 id 请求并映射列表', async () => {
		vi.mocked(api.listUserReactedMoments).mockResolvedValue({ code: 0, data: { items: [momentItem] } } as never);
		const r = await mount(() => 'u1');
		await waitFor(() => expect(r.loading).toBe(false));
		expect(api.listUserReactedMoments).toHaveBeenCalledTimes(1);
		expect(vi.mocked(api.listUserReactedMoments).mock.calls[0][0]).toBe('u1');
		expect(r.posts.map((p) => p.id)).toEqual(['m1']);
		expect(r.error).toBeNull();
	});

	it('无 id 时不发请求、直接 needsAuth（不拿空串换 10000）', async () => {
		const r = await mount(() => undefined);
		await waitFor(() => expect(r.loading).toBe(false));
		expect(api.listUserReactedMoments).not.toHaveBeenCalled();
		expect(r.needsAuth).toBe(true);
		expect(r.posts).toEqual([]);
	});

	it('10100 转 needsAuth', async () => {
		vi.mocked(api.listUserReactedMoments).mockResolvedValue({ code: 10100 } as never);
		const r = await mount(() => 'u1');
		await waitFor(() => expect(r.loading).toBe(false));
		expect(r.needsAuth).toBe(true);
	});
});
