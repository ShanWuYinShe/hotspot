// WS store 决策纯函数测试：登录态决定真实 WS 的连接/断开/等待。
import { describe, it, expect } from 'vitest';
import { resolveWsAction } from '@/stores/ws.svelte';

describe('resolveWsAction', () => {
	it('已登录连接、匿名断开、加载中等待', () => {
		expect(resolveWsAction('authenticated')).toBe('connect');
		expect(resolveWsAction('anonymous')).toBe('disconnect');
		expect(resolveWsAction('loading')).toBe('wait');
	});
});
