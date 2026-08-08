// TagPill 契约测试：可点击/静态形态、选中态类、点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import TagPill from '../TagPill.svelte';

describe('TagPill', () => {
	it('onClick 时渲染 button 并带 aria-pressed', () => {
		const { container } = render(TagPill, {
			props: { children: (() => '摄影') as never, onClick: () => {}, active: true }
		});
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn).not.toBeNull();
		expect(btn.getAttribute('aria-pressed')).toBe('true');
	});

	it('active 时应用选中态 token 类', () => {
		const { container } = render(TagPill, {
			props: { children: (() => '摄影') as never, onClick: () => {}, active: true }
		});
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn.className).toContain('bg-accent-soft');
		expect(btn.className).toContain('text-accent-default');
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		const { container } = render(TagPill, {
			props: { children: (() => '摄影') as never, onClick }
		});
		await fireEvent.click(container.querySelector('button') as Element);
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('无 onClick 时渲染静态 span', () => {
		const { container } = render(TagPill, { props: { children: (() => '标签') as never } });
		expect(container.querySelector('button')).toBeNull();
		expect(container.querySelector('span')).not.toBeNull();
	});
});
