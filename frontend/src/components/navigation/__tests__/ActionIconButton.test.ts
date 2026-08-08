// ActionIconButton 契约测试：aria、点击、disabled、variant 样式。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { ArrowLeft } from '@lucide/svelte';
import ActionIconButton from '../ActionIconButton.svelte';

describe('ActionIconButton', () => {
	it('渲染 aria-label 与 title', () => {
		render(ActionIconButton, { props: { icon: ArrowLeft, label: '返回' } });
		const btn = screen.getByRole('button', { name: '返回' });
		expect(btn.getAttribute('title')).toBe('返回');
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(ActionIconButton, { props: { icon: ArrowLeft, label: '返回', onClick } });
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('disabled 时按钮禁用并带禁用透明度类', () => {
		const { container } = render(ActionIconButton, {
			props: { icon: ArrowLeft, label: '返回', disabled: true }
		});
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn.disabled).toBe(true);
		expect(btn.className).toContain('disabled:opacity-50');
	});

	it('variant=overlay 使用遮罩背景 token', () => {
		const { container } = render(ActionIconButton, {
			props: { icon: ArrowLeft, label: '返回', variant: 'overlay' }
		});
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn.style.background).toContain('var(--cst-surface-overlay)');
	});
});
