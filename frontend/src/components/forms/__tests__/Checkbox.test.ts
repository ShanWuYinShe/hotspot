// Checkbox 契约测试：原生语义、label 关联、accent token、受控回调、disabled。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Checkbox from '../Checkbox.svelte';

describe('Checkbox', () => {
	it('渲染原生 checkbox 并带 ariaLabel', () => {
		render(Checkbox, { props: { checked: false, ariaLabel: '全选' } });
		const el = screen.getByLabelText('全选') as HTMLInputElement;
		expect(el.type).toBe('checkbox');
	});

	it('点击触发 onChange(checked)', async () => {
		const onChange = vi.fn();
		render(Checkbox, { props: { checked: false, onChange, ariaLabel: '同意' } });
		await fireEvent.click(screen.getByLabelText('同意'));
		expect(onChange).toHaveBeenCalled();
		expect(onChange.mock.calls[0][0]).toBe(true);
	});

	it('视觉走语义 token：16px + accent 主题色 + focus-ring', () => {
		render(Checkbox, { props: { checked: false, ariaLabel: '全选' } });
		const el = screen.getByLabelText('全选') as HTMLElement;
		expect(el.className).toContain('size-4');
		expect(el.className).toContain('accent-accent-default');
		expect(el.className).toContain('focus-ring');
	});

	it('disabled 时不可用', () => {
		render(Checkbox, { props: { checked: false, disabled: true, ariaLabel: '全选' } });
		expect((screen.getByLabelText('全选') as HTMLInputElement).disabled).toBe(true);
	});
});
