// createOptimisticFollow 契约测试：乐观切换、失败回滚（code!==0 与网络异常）、防重入。
// 工厂内无 $effect，可直接在测试中实例化。
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@/api', () => ({
	follow: vi.fn()
}));
vi.mock('@/stores/toast.svelte', () => ({
	showToast: vi.fn()
}));

import { follow } from '@/api';
import { showToast } from '@/stores/toast.svelte';
import { createOptimisticFollow } from './relationships.svelte';

const mockedFollow = vi.mocked(follow);
const mockedShowToast = vi.mocked(showToast);

describe('createOptimisticFollow', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('成功切换：isFollowed 翻转，请求结束后清除 pending', async () => {
		mockedFollow.mockResolvedValue({ code: 0, data: undefined });
		const state = createOptimisticFollow();

		expect(state.isFollowed('u1', false)).toBe(false);
		const p = state.toggle('u1', false);
		expect(state.isPending('u1')).toBe(true);
		const r = await p;

		expect(r).toEqual({ ok: true, followed: true });
		expect(mockedFollow).toHaveBeenCalledWith({ user_id: 'u1', active: true });
		expect(state.isFollowed('u1', false)).toBe(true);
		expect(state.isPending('u1')).toBe(false);
	});

	it('响应 code!==0 视为失败：回滚到基准并 toast（首页版历史语义漂移的收敛点）', async () => {
		mockedFollow.mockResolvedValue({ code: 1, message: '不能关注自己', data: undefined });
		const state = createOptimisticFollow();

		const r = await state.toggle('u1', false);

		expect(r).toEqual({ ok: false, followed: false });
		expect(state.isFollowed('u1', false)).toBe(false);
		expect(mockedShowToast).toHaveBeenCalledWith('操作失败，请重试', 'danger');
	});

	it('网络异常：回滚并 toast', async () => {
		mockedFollow.mockRejectedValue(new Error('network down'));
		const state = createOptimisticFollow();

		const r = await state.toggle('u1', true);

		expect(r).toEqual({ ok: false, followed: true });
		expect(state.isFollowed('u1', true)).toBe(true);
		expect(mockedShowToast).toHaveBeenCalledWith('操作失败，请重试', 'danger');
	});

	it('pending 期间重复 toggle 被拦截，不发起第二次请求', async () => {
		let resolveFollow!: (v: { code: number; data: undefined }) => void;
		mockedFollow.mockImplementation(
			() => new Promise<{ code: number; data: undefined }>((resolve) => (resolveFollow = resolve))
		);
		const state = createOptimisticFollow();

		const first = state.toggle('u1', false);
		const second = await state.toggle('u1', false);
		resolveFollow({ code: 0, data: undefined });
		const r = await first;

		expect(second).toEqual({ ok: false, followed: false });
		expect(mockedFollow).toHaveBeenCalledTimes(1);
		expect(r).toEqual({ ok: true, followed: true });
	});
});
