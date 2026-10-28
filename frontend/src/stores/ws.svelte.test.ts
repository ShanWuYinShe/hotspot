// WS store 决策纯函数测试：mock 模式必须跳过真实 WS（否则 4001 刷新重连循环）。
import { describe, it, expect } from 'vitest';
import { resolveWsAction } from '@/stores/ws.svelte';

describe('resolveWsAction', () => {
	it('mock 启用时一律断开：MSW 不能模拟 WS，mock 登录态连真实后端必被 4001 关闭', () => {
		expect(resolveWsAction('authenticated', true)).toBe('disconnect');
		expect(resolveWsAction('anonymous', true)).toBe('disconnect');
		expect(resolveWsAction('loading', true)).toBe('disconnect');
	});

	it('非 mock：登录态连接、匿名断开、加载中等待', () => {
		expect(resolveWsAction('authenticated', false)).toBe('connect');
		expect(resolveWsAction('anonymous', false)).toBe('disconnect');
		expect(resolveWsAction('loading', false)).toBe('wait');
	});
});