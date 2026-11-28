import { describe, expect, it } from 'vitest';
import { DEFAULT_RECALL_WINDOW_MS, inRecallWindow, resolveRecallWindowMs } from './chat.svelte';

describe('recall window', () => {
	it('未配置时回退到后端默认（120s）', () => {
		expect(resolveRecallWindowMs()).toBe(DEFAULT_RECALL_WINDOW_MS);
		expect(DEFAULT_RECALL_WINDOW_MS).toBe(120_000);
	});

	it('配置值按秒换算为毫秒', () => {
		expect(resolveRecallWindowMs(300)).toBe(300_000);
	});

	it('0 表示不允许撤回', () => {
		expect(resolveRecallWindowMs(0)).toBe(0);
		expect(inRecallWindow(new Date().toISOString(), 0)).toBe(false);
	});

	it('窗口内可撤回、超出不可撤回', () => {
		const now = Date.now();
		expect(inRecallWindow(new Date(now - 60_000).toISOString(), 120_000)).toBe(true);
		expect(inRecallWindow(new Date(now - 180_000).toISOString(), 120_000)).toBe(false);
	});

	it('无时间戳放行（服务端兜底）', () => {
		expect(inRecallWindow(undefined, 120_000)).toBe(true);
	});
});
