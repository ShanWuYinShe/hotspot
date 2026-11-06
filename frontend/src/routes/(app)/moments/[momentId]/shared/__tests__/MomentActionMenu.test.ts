import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MomentActionMenu from '../MomentActionMenu.svelte';

describe('MomentActionMenu 动态操作菜单', () => {
	it('访客视角：包含复制链接、刷新动态、举报动态，不含编辑与删除', async () => {
		const onCopyLink = vi.fn();
		const onReload = vi.fn();
		const onReport = vi.fn();
		const onEdit = vi.fn();
		const onDelete = vi.fn();

		render(MomentActionMenu, {
			props: {
				isAuthor: false,
				onCopyLink,
				onReload,
				onReport,
				onEdit,
				onDelete
			}
		});

		const trigger = screen.getByRole('button', { name: '更多操作' });
		expect(trigger).toBeTruthy();
		await fireEvent.click(trigger);

		expect(screen.getByText('复制链接')).toBeTruthy();
		expect(screen.getByText('刷新动态')).toBeTruthy();
		expect(screen.getByText('举报动态')).toBeTruthy();
		expect(screen.queryByText('编辑动态')).toBeNull();
		expect(screen.queryByText('删除动态')).toBeNull();

		await fireEvent.click(screen.getByText('复制链接'));
		expect(onCopyLink).toHaveBeenCalledTimes(1);
	});

	it('作者视角：包含编辑动态与删除动态', async () => {
		const onCopyLink = vi.fn();
		const onReload = vi.fn();
		const onEdit = vi.fn();
		const onDelete = vi.fn();

		render(MomentActionMenu, {
			props: {
				isAuthor: true,
				onCopyLink,
				onReload,
				onEdit,
				onDelete
			}
		});

		const trigger = screen.getByRole('button', { name: '更多操作' });
		await fireEvent.click(trigger);

		expect(screen.getByText('编辑动态')).toBeTruthy();
		expect(screen.getByText('删除动态')).toBeTruthy();
		expect(screen.queryByText('举报动态')).toBeNull();

		await fireEvent.click(screen.getByText('编辑动态'));
		expect(onEdit).toHaveBeenCalledTimes(1);
	});
});
