// Toggle 契约测试：role=switch、aria-label、点击触发回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Toggle from '../Toggle.svelte';

describe('Toggle', () => {
	it('渲染 role=switch 与 aria-label', () => {
		render(Toggle, { props: { on: false, onToggle: () => {}, label: '深色模式' } });
		expect(screen.getByRole('switch', { name: '深色模式' })).toBeTruthy();
	});

	it('点击触发 onToggle', async () => {
		const onToggle = vi.fn();
		render(Toggle, { props: { on: false, onToggle, label: '通知' } });
		await fireEvent.click(screen.getByRole('switch'));
		expect(onToggle).toHaveBeenCalledTimes(1);
	});
});
