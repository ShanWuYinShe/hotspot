import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SheetShellHost from './SheetShellHost.svelte';

describe('SheetShell 下拉拖拽关闭手势契约（P0-3）', () => {
	beforeEach(() => {
		Object.defineProperty(window, 'innerWidth', { configurable: true, value: 375 });
		Object.defineProperty(window, 'innerHeight', { configurable: true, value: 812 });
	});

	it('下拉超过阈值时顺势向下滑出（不弹回 0），延时触发 onClose', async () => {
		vi.useFakeTimers();
		const onClose = vi.fn();

		render(SheetShellHost, {
			props: { title: '测试弹层', onClose }
		});

		const handle = document.querySelector('[style*="cursor:grab"]') as HTMLElement;
		expect(handle).toBeTruthy();

		// 模拟拖拽
		handle.setPointerCapture = vi.fn();
		fireEvent.pointerDown(handle, { clientY: 100, pointerId: 1, pointerType: 'touch' });
		fireEvent.pointerMove(handle, { clientY: 200, pointerId: 1, pointerType: 'touch' }); // 下移 100px

		const shell = handle.closest('.flex.h-full.flex-col') as HTMLElement;
		expect(shell.style.transform).toBe('translateY(100px)');

		// 松手
		fireEvent.pointerUp(handle, { pointerId: 1 });

		// 验证：不应该弹回 0，而是顺势向下滑出
		expect(shell.style.transform).not.toBe('translateY(0px)');
		expect(shell.style.transform).toContain('translateY(');
		expect(shell.style.transition).toContain('var(--cst-motion-rapid)');

		// 定时器到期触发 onClose
		expect(onClose).not.toHaveBeenCalled();
		vi.advanceTimersByTime(140);
		expect(onClose).toHaveBeenCalledOnce();

		vi.useRealTimers();
	});

	it('下拉未达阈值松手时，弹性平滑复位至 0', async () => {
		const onClose = vi.fn();

		render(SheetShellHost, {
			props: { title: '测试弹层', onClose }
		});

		const handle = document.querySelector('[style*="cursor:grab"]') as HTMLElement;
		expect(handle).toBeTruthy();
		handle.setPointerCapture = vi.fn();
		fireEvent.pointerDown(handle, { clientY: 100, pointerId: 1, pointerType: 'touch' });
		fireEvent.pointerMove(handle, { clientY: 120, pointerId: 1, pointerType: 'touch' }); // 仅 20px

		const shell = handle.closest('.flex.h-full.flex-col') as HTMLElement;
		expect(shell.style.transform).toBe('translateY(20px)');

		// 松手
		fireEvent.pointerUp(handle, { pointerId: 1 });

		// 验证：未达阈值复位且带 bounce 曲线
		expect(shell.style.transform).toBe('translateY(0px)');
		expect(shell.style.transition).toContain('var(--cst-ease-bounce)');
		expect(onClose).not.toHaveBeenCalled();
	});
});
