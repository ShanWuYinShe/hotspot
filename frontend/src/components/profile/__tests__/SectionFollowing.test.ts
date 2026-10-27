// SectionFollowing 契约测试：取关后行立即移除 + 计数同步 + 空态可达；
// 失败回滚后行恢复（回归：此前残留变灰行、计数不变、空态切页内不可达）。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';

const mockRel = vi.hoisted(() => ({
	following: [
		{ id: 'u1', name: '甲', avatar: 'https://example.com/a1.png', desc: '', isMutual: false },
		{ id: 'u2', name: '乙', avatar: 'https://example.com/a2.png', desc: '', isMutual: true }
	]
}));

vi.mock('@/lib/profile-data.svelte', () => ({
	useRelationships: () => ({
		following: mockRel.following,
		loading: false,
		error: null,
		needsAuth: false,
		retry: vi.fn()
	})
}));

vi.mock('@/api', () => ({
	follow: vi.fn(async () => ({ code: 0 }))
}));

vi.mock('@/stores/toast.svelte', () => ({
	showToast: vi.fn()
}));

import SectionFollowing from '../SectionFollowing.svelte';
import { follow } from '@/api';
import { showToast } from '@/stores/toast.svelte';

const followMock = vi.mocked(follow);
const toastMock = vi.mocked(showToast);

beforeEach(() => {
	vi.clearAllMocks();
	followMock.mockResolvedValue({ code: 0 } as never);
	mockRel.following = [
		{ id: 'u1', name: '甲', avatar: 'https://example.com/a1.png', desc: '', isMutual: false },
		{ id: 'u2', name: '乙', avatar: 'https://example.com/a2.png', desc: '', isMutual: true }
	];
});

describe('SectionFollowing', () => {
	it('渲染两行与计数 2 人', () => {
		render(SectionFollowing);
		// Avatar 占位层会同时渲染首字母（aria-hidden）；断言行存在用 getAllByText
		expect(screen.getAllByText('甲').length).toBeGreaterThan(0);
		expect(screen.getAllByText('乙').length).toBeGreaterThan(0);
		expect(screen.getByText('2 人')).toBeTruthy();
	});

	it('取关成功：行立即移除、计数同步为 1 人', async () => {
		render(SectionFollowing);
		const btns = screen.getAllByRole('button', { name: '已关注' });
		expect(btns).toHaveLength(2);
		await fireEvent.click(btns[0]);
		await waitFor(() => expect(screen.queryAllByText('甲')).toHaveLength(0));
		expect(screen.getAllByText('乙').length).toBeGreaterThan(0);
		expect(screen.getByText('1 人')).toBeTruthy();
		expect(followMock).toHaveBeenCalledWith({ user_id: 'u1', active: false });
	});

	it('取关全部：空态可达', async () => {
		render(SectionFollowing);
		for (const b of screen.getAllByRole('button', { name: '已关注' })) {
			await fireEvent.click(b);
		}
		await waitFor(() => expect(screen.getByText('还没有关注')).toBeTruthy());
		expect(screen.getByText('0 人')).toBeTruthy();
	});

	it('取关失败：行恢复 + 失败 toast', async () => {
		followMock.mockResolvedValue({ code: 10400, message: 'x' } as never);
		render(SectionFollowing);
		await fireEvent.click(screen.getAllByRole('button', { name: '已关注' })[0]);
		await waitFor(() => expect(toastMock).toHaveBeenCalledWith('操作失败，请重试', 'danger'));
		expect(screen.getAllByText('甲').length).toBeGreaterThan(0);
		expect(screen.getByText('2 人')).toBeTruthy();
	});
});
