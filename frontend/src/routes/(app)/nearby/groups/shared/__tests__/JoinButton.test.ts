// JoinButton 契约测试：加入/已加入形态与切换回调。
// 双态样式已类化（inline background 会压制 hover 反馈），断言走类名。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import JoinButton from '../JoinButton.svelte';

describe('JoinButton', () => {
	it('未加入时渲染「加入」与 accent 实底类', () => {
		const { container } = render(JoinButton, {
			props: { group: { joined: false } as never, onToggle: () => {} }
		});
		expect(screen.getByRole('button', { name: '加入' })).toBeTruthy();
		expect((container.querySelector('button') as HTMLElement).className).toContain('bg-accent-default');
		expect((container.querySelector('button') as HTMLElement).getAttribute('aria-pressed')).toBe('false');
	});

	it('已加入时渲染「已加入」与描边类', () => {
		const { container } = render(JoinButton, {
			props: { group: { joined: true } as never, onToggle: () => {} }
		});
		expect(screen.getByRole('button', { name: /已加入/ })).toBeTruthy();
		expect((container.querySelector('button') as HTMLElement).className).toContain('border-border-strong');
		expect((container.querySelector('button') as HTMLElement).getAttribute('aria-pressed')).toBe('true');
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
