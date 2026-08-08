// JoinButton 契约测试：加入/已加入形态与切换回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import JoinButton from '../JoinButton.svelte';

describe('JoinButton', () => {
	it('未加入时渲染「加入」与 accent 样式', () => {
		const { container } = render(JoinButton, {
			props: { group: { joined: false } as never, onToggle: () => {} }
		});
		expect(screen.getByRole('button', { name: '加入' })).toBeTruthy();
		expect((container.querySelector('button') as HTMLElement).style.background).toContain('var(--cst-accent-default)');
	});

	it('已加入时渲染「已加入」与描边样式', () => {
		const { container } = render(JoinButton, {
			props: { group: { joined: true } as never, onToggle: () => {} }
		});
		expect(screen.getByRole('button', { name: /已加入/ })).toBeTruthy();
		expect((container.querySelector('button') as HTMLElement).style.border).toContain('var(--cst-border-strong)');
	});

	it('点击触发 onToggle', async () => {
		const onToggle = vi.fn();
		render(JoinButton, {
			props: { group: { joined: false } as never, onToggle }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onToggle).toHaveBeenCalledTimes(1);
	});
});
