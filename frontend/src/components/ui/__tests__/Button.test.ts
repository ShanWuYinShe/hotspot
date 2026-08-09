// Button 契约测试：变体/尺寸类、loading 状态、aria-busy、事件透传。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Button from '../button.svelte';

const label = (() => '保存') as never;

describe('Button', () => {
	it('默认渲染 primary md 按钮', () => {
		const { container } = render(Button, { props: { children: label } });
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn).not.toBeNull();
		expect(btn.className).toContain('bg-accent-default');
		expect(btn.className).toContain('text-text-on-accent');
		expect(btn.className).toContain('h-10');
		expect(btn.type).toBe('button');
	});

	it('loading 时设置 aria-busy 并禁用交互', () => {
		const { container } = render(Button, { props: { children: label, loading: true } });
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn.getAttribute('aria-busy')).toBe('true');
		expect(btn.className).toContain('pointer-events-none');
	});

	it('danger 变体使用状态色与 on-* 文字 token', () => {
		const { container } = render(Button, { props: { children: label, variant: 'danger' } });
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn.className).toContain('bg-status-danger');
		expect(btn.className).toContain('text-text-on-status');
	});

	it('点击触发事件并透传原生属性', async () => {
		const onClick = vi.fn();
		const { container } = render(Button, {
			props: { children: label, onclick: onClick, 'aria-label': '保存设置' }
		});
		const btn = container.querySelector('button') as HTMLButtonElement;
		expect(btn.getAttribute('aria-label')).toBe('保存设置');
		await fireEvent.click(btn);
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});
