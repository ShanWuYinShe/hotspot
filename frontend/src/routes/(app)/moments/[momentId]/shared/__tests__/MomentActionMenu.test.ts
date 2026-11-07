import { afterEach, describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MomentActionMenu from '../MomentActionMenu.svelte';

// bits-ui body-scroll-lock 在最后一个锁释放后会安排 ~24ms 的延迟清理定时器；
// 若定时器在本文件结束、jsdom 环境拆除后才触发，会抛
// "document is not defined" 并拖红整轮 CI（同 ConfirmDialog.test.ts 的处置）。
afterEach(async () => {
	await new Promise((r) => setTimeout(r, 60));
});

describe('MomentActionMenu 动态操作菜单', () => {
	it('访客视角：缺省不包含复制链接（已有独立外置按钮），包含刷新动态与举报动态', async () => {
		const onReload = vi.fn();
		const onReport = vi.fn();
		const onEdit = vi.fn();
		const onDelete = vi.fn();

		render(MomentActionMenu, {
			props: {
				isAuthor: false,
				onReload,
				onReport,
				onEdit,
				onDelete
			}
		});

		const trigger = screen.getByRole('button', { name: '更多操作' });
		expect(trigger).toBeTruthy();
		await fireEvent.click(trigger);

		expect(screen.queryByText('复制链接')).toBeNull();
		expect(screen.getByText('刷新动态')).toBeTruthy();
		expect(screen.getByText('举报动态')).toBeTruthy();
		expect(screen.queryByText('编辑动态')).toBeNull();
		expect(screen.queryByText('删除动态')).toBeNull();

		await fireEvent.click(screen.getByText('刷新动态'));
		expect(onReload).toHaveBeenCalledTimes(1);
	});

	it('显式提供 onCopyLink 时渲染复制链接项', async () => {
		const onCopyLink = vi.fn();

		render(MomentActionMenu, {
			props: {
				isAuthor: false,
				onCopyLink
			}
		});

		const trigger = screen.getByRole('button', { name: '更多操作' });
		await fireEvent.click(trigger);

		expect(screen.getByText('复制链接')).toBeTruthy();
		await fireEvent.click(screen.getByText('复制链接'));
		expect(onCopyLink).toHaveBeenCalledTimes(1);
	});

	it('作者视角：包含编辑动态与删除动态', async () => {
		const onReload = vi.fn();
		const onEdit = vi.fn();
		const onDelete = vi.fn();

		render(MomentActionMenu, {
			props: {
				isAuthor: true,
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
