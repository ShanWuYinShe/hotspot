// Textarea 契约测试：壳/label/错误行/受控回调/rows/maxlength/disabled/resize。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Textarea from '../Textarea.svelte';

describe('Textarea', () => {
	it('渲染多行输入与 placeholder', () => {
		render(Textarea, { props: { value: '', onChange: () => {}, placeholder: '介绍一下自己…' } });
		expect(screen.getByPlaceholderText('介绍一下自己…')).toBeTruthy();
	});

	it('输入触发 onChange', async () => {
		const onChange = vi.fn();
		render(Textarea, { props: { value: '', onChange } });
		await fireEvent.input(screen.getByRole('textbox'), { target: { value: '你好' } });
		expect(onChange).toHaveBeenCalledWith('你好');
	});

	it('label 与 textarea 以 id 关联', () => {
		render(Textarea, { props: { value: '', onChange: () => {}, label: '简介' } });
		expect(screen.getByLabelText('简介')).toBeTruthy();
	});

	it('壳统一 px-3.5 py-3（所有多行输入同一表面）', () => {
		const { container } = render(Textarea, { props: { value: '', onChange: () => {}, ariaLabel: '简介' } });
		const surface = container.querySelector('.input-surface') as HTMLElement;
		expect(surface.className).toContain('px-3.5');
		expect(surface.className).toContain('py-3');
	});

	it('rows/maxlength/disabled 透传', () => {
		render(Textarea, {
			props: { value: '', onChange: () => {}, ariaLabel: '简介', rows: 5, maxlength: 120, disabled: true }
		});
		const el = screen.getByLabelText('简介') as HTMLTextAreaElement;
		expect(el.rows).toBe(5);
		expect(el.maxLength).toBe(120);
		expect(el.disabled).toBe(true);
	});

	it('错误态：aria-invalid + alert 行 + aria-describedby', () => {
		render(Textarea, {
			props: { value: '', onChange: () => {}, label: '简介', hasError: true, errorText: '最多 120 字' }
		});
		const el = screen.getByLabelText('简介') as HTMLTextAreaElement;
		const alert = screen.getByRole('alert');
		expect(el.getAttribute('aria-invalid')).toBe('true');
		expect(el.getAttribute('aria-describedby')).toBe(alert.id);
		expect(alert.textContent).toBe('最多 120 字');
	});

	it('文本域无自身 resize 手柄（默认 none）', () => {
		const { container } = render(Textarea, { props: { value: '', onChange: () => {}, ariaLabel: '简介' } });
		const el = container.querySelector('textarea') as HTMLTextAreaElement;
		expect(el.className).toContain('resize-none');
	});
});
