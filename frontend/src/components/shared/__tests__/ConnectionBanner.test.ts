// ConnectionBanner 契约测试：断线提示的显示条件（mock stores）。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import ConnectionBanner from '../ConnectionBanner.svelte';

const wsState = { status: 'connecting' };
const authState = { isAuthenticated: false };

vi.mock('@/stores/ws.svelte', () => ({
	ws: { get status() { return wsState.status; } }
}));
vi.mock('@/stores/auth.svelte', () => ({
	auth: { get isAuthenticated() { return authState.isAuthenticated; } }
}));

describe('ConnectionBanner', () => {
	beforeEach(() => {
		wsState.status = 'connecting';
		authState.isAuthenticated = false;
	});

	it('未认证时不渲染', async () => {
		render(ConnectionBanner, { props: {} });
		expect(screen.queryByText(/网络连接已断开/)).toBeNull();
	});

	it('认证但从未成功连接时不渲染', async () => {
		authState.isAuthenticated = true;
		render(ConnectionBanner, { props: {} });
		expect(screen.queryByText(/网络连接已断开/)).toBeNull();
	});
});
