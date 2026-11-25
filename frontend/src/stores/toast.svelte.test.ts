// Toast store 契约测试：添加/自动消失/手动关闭。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { showToast, dismissToast, toast } from './toast.svelte';

describe('toast store', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		// 模块级单例：清掉上一用例残留（新增行动入口用例不会走定时器路径）。
		for (const item of [...toast.items]) dismissToast(item.id);
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('showToast 添加条目并在超时后自动消失', () => {
		showToast('已加入群组', 'success');
		expect(toast.items).toHaveLength(1);
		expect(toast.items[0]).toMatchObject({ message: '已加入群组', kind: 'success' });
		vi.advanceTimersByTime(3500);
		expect(toast.items).toHaveLength(0);
	});

	it('showToast 可携带可选行动入口', () => {
		const onAction = vi.fn();
		showToast('分享链接已复制', 'success', { label: '管理分享', onAction });
		expect(toast.items[0].action).toMatchObject({ label: '管理分享' });
		toast.items[0].action?.onAction();
		expect(onAction).toHaveBeenCalledTimes(1);
	});

	it('dismissToast 手动关闭指定条目', () => {
		showToast('通知');
		const id = toast.items[0].id;
		dismissToast(id);
		expect(toast.items).toHaveLength(0);
	});
});
