// ConfirmDialog 契约测试：标题/按钮渲染与确认/取消回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ConfirmDialog from '../ConfirmDialog.svelte';

describe('ConfirmDialog', () => {
	it('渲染标题与确认/取消按钮', () => {
		render(ConfirmDialog, {
			props: { open: true, title: '删除这条动态？', confirmLabel: '确认删除', onCancel: () => {}, onConfirm: () => {} }
		});
		expect(screen.getByText('删除这条动态？')).toBeTruthy();
		expect(screen.getByRole('button', { name: '确认删除' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '取消' })).toBeTruthy();
	});

	it('确认按钮触发 onConfirm', async () => {
		const onConfirm = vi.fn();
		render(ConfirmDialog, {
			props: { open: true, title: '退出登录？', confirmLabel: '退出登录', onCancel: () => {}, onConfirm }
		});
		await fireEvent.click(screen.getByRole('button', { name: '退出登录' }));
		expect(onConfirm).toHaveBeenCalledTimes(1);
	});

	it('取消按钮触发 onCancel', async () => {
		const onCancel = vi.fn();
		render(ConfirmDialog, {
			props: { open: true, title: '退出群组？', confirmLabel: '确认退出', onCancel, onConfirm: () => {} }
		});
		await fireEvent.click(screen.getByRole('button', { name: '取消' }));
		expect(onCancel).toHaveBeenCalledTimes(1);
	});
});
