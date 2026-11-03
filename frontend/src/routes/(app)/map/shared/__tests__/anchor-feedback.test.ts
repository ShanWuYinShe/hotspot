import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createAnchorFeedback } from '../anchor-feedback.svelte';

describe('anchor-feedback 状态机', () => {
	beforeEach(() => vi.useFakeTimers());
	afterEach(() => vi.useRealTimers());

	it('成功：pending → success(1.4s) → idle，按钮恢复主按钮', () => {
		const fb = createAnchorFeedback();
		fb.run(() => true);
		expect(fb.state).toBe('pending');
		vi.advanceTimersByTime(1000);
		expect(fb.state).toBe('success');
		vi.advanceTimersByTime(1400);
		expect(fb.state).toBe('idle');
	});

	it('失败：action 返回 false → error(1.8s) → idle', () => {
		const fb = createAnchorFeedback();
		fb.run(() => false);
		vi.advanceTimersByTime(1000);
		expect(fb.state).toBe('error');
		vi.advanceTimersByTime(1800);
		expect(fb.state).toBe('idle');
	});

	it('action 抛错也按失败处理（不把异常抛到界面）', () => {
		const fb = createAnchorFeedback();
		fb.run(() => {
			throw new Error('storage unavailable');
		});
		vi.advanceTimersByTime(1000);
		expect(fb.state).toBe('error');
	});

	it('pending 中重复点击被忽略；reset/destroy 取消定时器', () => {
		const fb = createAnchorFeedback();
		const action = vi.fn(() => true);
		fb.run(action);
		fb.run(action);
		expect(action).toHaveBeenCalledTimes(1);
		fb.destroy();
		vi.advanceTimersByTime(3000);
		expect(fb.state).toBe('pending'); // 定时器已取消，状态停在 pending（组件卸载后不再关心）
	});
});

