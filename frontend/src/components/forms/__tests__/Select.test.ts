// Select 契约测试：field/ghost 两变体、label 关联、密度、chevron、disabled。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Select from '../Select.svelte';

describe('Select', () => {
	it('field 变体：label 关联 + input-surface + 16 圆角', () => {
		render(Select, { props: { value: '', onChange: () => {}, label: '类型' } });
		const el = screen.getByLabelText('类型') as HTMLSelectElement;
		expect(el.tagName).toBe('SELECT');
		expect(el.className).toContain('input-surface');
		expect(el.className).toContain('rounded-md');
		expect(el.className).toContain('appearance-none');
	});

	it('field 默认 43px 档（px-4 py-2.5），compact 为 39px 档（px-3.5 py-2）', () => {
		const { container, unmount } = render(Select, {
			props: { value: '', onChange: () => {}, ariaLabel: '类型' }
		});
		let el = container.querySelector('select') as HTMLElement;
		expect(el.className).toContain('px-4');
		expect(el.className).toContain('py-2.5');
		unmount();
		const second = render(Select, {
			props: { value: '', onChange: () => {}, ariaLabel: '类型', density: 'compact' }
		});
		el = second.container.querySelector('select') as HTMLElement;
		expect(el.className).toContain('px-3.5');
		expect(el.className).toContain('py-2');
	});

	it('field 变体渲染 chevron（原生箭头隐藏后由组件绘制）', () => {
		const { container } = render(Select, { props: { value: '', onChange: () => {}, ariaLabel: '类型' } });
		expect(container.querySelector('svg')).toBeTruthy();
	});

	it('ghost 变体：透明底 + accent 文字 + 无 input-surface（行内排序）', () => {
		const { container } = render(Select, {
			props: { value: '距离', onChange: () => {}, variant: 'ghost', ariaLabel: '排序方式' }
		});
		const el = container.querySelector('select') as HTMLElement;
		expect(el.className).not.toContain('input-surface');
		expect(el.className).toContain('bg-transparent');
		expect(el.className).toContain('text-accent-default');
	});

	it('disabled 透传', () => {
		render(Select, { props: { value: '', onChange: () => {}, ariaLabel: '类型', disabled: true } });
		expect((screen.getByLabelText('类型') as HTMLSelectElement).disabled).toBe(true);
	});
});
