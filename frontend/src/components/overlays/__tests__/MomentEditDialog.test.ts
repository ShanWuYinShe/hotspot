// MomentEditDialog 契约测试：可见性未知时禁改且提交省略 privacy_level（防私有动态被误存为公开）。
import { afterEach, describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import MomentEditDialog from '../MomentEditDialog.svelte';
import { updateMoment } from '@/api';

vi.mock('@/api', () => ({ updateMoment: vi.fn() }));
vi.mock('@/stores/toast.svelte', () => ({ showToast: vi.fn() }));

const BASE = { description: '旧内容', tags: ['咖啡'] };

// bits-ui body-scroll-lock 在最后一个锁释放后会安排 ~24ms 的延迟清理定时器；
// 慢 CI 上定时器可能在本文件结束、jsdom 环境拆除后才触发，抛
// "document is not defined" 并拖红整轮 CI。afterEach 等一拍让清理
// 在环境存活期内落地（实测默认延迟 24ms，60ms 留足余量）。
afterEach(async () => {
	await new Promise((r) => setTimeout(r, 60));
});

describe('MomentEditDialog 可见性门控', () => {
	beforeEach(() => {
		vi.mocked(updateMoment).mockReset();
		vi.mocked(updateMoment).mockResolvedValue({ code: 0, data: {} as never });
	});

	it('可见性未探测完成时禁用选项，提交省略 privacy_level（保持原值）', async () => {
		render(MomentEditDialog, {
			props: { open: true, momentId: 'm1', initial: { ...BASE, privacy_level: null }, onClose: () => {}, onUpdated: () => {} }
		});
		expect(screen.getByRole('button', { name: '公开可见' })).toHaveProperty('disabled', true);
		expect(screen.getByRole('button', { name: '仅自己' })).toHaveProperty('disabled', true);
		expect(screen.getByText('可见性获取中，稍候可修改；现在保存将保持当前设置')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '保存' }));
		await waitFor(() => {
			expect(updateMoment).toHaveBeenCalledWith(
				expect.not.objectContaining({ privacy_level: expect.anything() })
			);
		});
	});

	it('可见性已知时正常回填并随提交发送', async () => {
		render(MomentEditDialog, {
			props: { open: true, momentId: 'm1', initial: { ...BASE, privacy_level: 'private' }, onClose: () => {}, onUpdated: () => {} }
		});
		expect(screen.getByRole('button', { name: '仅自己' })).toHaveProperty('disabled', false);
		expect(screen.getByRole('button', { name: '仅自己' }).getAttribute('aria-pressed')).toBe('true');
		await fireEvent.click(screen.getByRole('button', { name: '保存' }));
		await waitFor(() => {
			expect(updateMoment).toHaveBeenCalledWith(expect.objectContaining({ privacy_level: 'private' }));
		});
	});
});
