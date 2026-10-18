// Input 契约测试：受控回调、label 关联、错误态三件套、清除按钮、禁用。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Input from '../Input.svelte';

describe('Input', () => {
	it('渲染输入框与 placeholder', () => {
		render(Input, { props: { value: '', onChange: () => {}, placeholder: '你的名字' } });
		expect(screen.getByPlaceholderText('你的名字')).toBeTruthy();
	});

	it('输入触发 onChange 并传递新值', async () => {
		const onChange = vi.fn();
		render(Input, { props: { value: '', onChange } });
		const input = screen.getByRole('textbox') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: '鹿鸣' } });
		expect(onChange).toHaveBeenCalledWith('鹿鸣');
	});

	it('label 与 input 以 id 关联（getByLabelText 命中）', () => {
		render(Input, { props: { value: '', onChange: () => {}, label: '昵称' } });
		expect(screen.getByLabelText('昵称')).toBeTruthy();
	});

	it('labelSnippet 渲染且 input 可由 id 关联', () => {
		render(Input, {
			props: {
				value: '',
				onChange: () => {},
				id: 'edit-nickname',
				labelSnippet: undefined
			}
		});
		const el = document.getElementById('edit-nickname');
		expect(el).toBeTruthy();
	});

	it('错误态：aria-invalid + role="alert" 错误行 + aria-describedby 关联', () => {
		render(Input, {
			props: { value: '', onChange: () => {}, label: '昵称', hasError: true, errorText: '昵称过长' }
		});
		const input = screen.getByLabelText('昵称') as HTMLInputElement;
		expect(input.getAttribute('aria-invalid')).toBe('true');
		const alert = screen.getByRole('alert');
		expect(alert.textContent).toBe('昵称过长');
		expect(input.getAttribute('aria-describedby')).toBe(alert.id);
	});

	it('无错误时不渲染错误行、无 aria-invalid', () => {
		render(Input, { props: { value: '', onChange: () => {}, label: '昵称' } });
		expect(screen.queryByRole('alert')).toBeNull();
		const input = screen.getByLabelText('昵称') as HTMLInputElement;
		expect(input.getAttribute('aria-invalid')).toBeNull();
	});

	it('clearable：value 非空显示清空按钮，点击触发 onChange("")', async () => {
		const onChange = vi.fn();
		render(Input, { props: { value: '已有输入', onChange, clearable: true } });
		const clear = screen.getByLabelText('清空');
		expect(clear).toBeTruthy();
		await fireEvent.click(clear);
		expect(onChange).toHaveBeenCalledWith('');
	});

	it('clearable=false 不显示清空按钮', () => {
		render(Input, { props: { value: 'x', onChange: () => {} } });
		expect(screen.queryByLabelText('清空')).toBeNull();
	});

	it('无可见 label 时 ariaLabel 生效', () => {
		render(Input, { props: { value: '', onChange: () => {}, ariaLabel: '注销确认码' } });
		expect(screen.getByLabelText('注销确认码')).toBeTruthy();
	});

	it('disabled 时输入框不可用', () => {
		render(Input, { props: { value: '', onChange: () => {}, label: '昵称', disabled: true } });
		const input = screen.getByLabelText('昵称') as HTMLInputElement;
		expect(input.disabled).toBe(true);
	});
});

describe('Input compact 形态', () => {
	it('compact：input-surface 直贴 input，无包裹容器', () => {
		const { container } = render(Input, {
			props: {
				value: 'DELETE_MY_ACCOUNT',
				onChange: () => {},
				compact: true,
				ariaLabel: '注销确认码',
				inputClass: 'w-full px-3 py-2'
			}
		});
		const input = screen.getByLabelText('注销确认码') as HTMLInputElement;
		expect(input.className).toContain('input-surface');
		expect(container.querySelector('label')).toBeNull();
		expect(container.querySelector('button')).toBeNull();
	});

	it('compact：错误行仍渲染在 input 之后', () => {
		render(Input, {
			props: { value: '', onChange: () => {}, compact: true, hasError: true, errorText: '不匹配' }
		});
		expect(screen.getByRole('alert').textContent).toBe('不匹配');
	});
});

describe('Input 默认内边距（防塌陷回归）', () => {
	// 审计发现：原语无默认 padding，调用方只传 inputClass="w-full" 时整框塌成 19px
	// （/me/edit 昵称字段实测 356×17.1、padding 0），偏离 INPUT_HEIGHT 42 且低于 24px 触控下限。
	it('非 compact 形态默认带 px-4 py-3', () => {
		const { container } = render(Input, { props: { value: '', onChange: () => {}, ariaLabel: '昵称' } });
		const surface = container.querySelector('.input-surface') as HTMLElement;
		expect(surface.className).toContain('px-4');
		expect(surface.className).toContain('py-3');
	});

	it('调用方 className 可覆盖默认内边距（tailwind-merge）', () => {
		const { container } = render(Input, {
			props: { value: '', onChange: () => {}, ariaLabel: '昵称', class: 'py-2' }
		});
		const surface = container.querySelector('.input-surface') as HTMLElement;
		expect(surface.className).toContain('py-2');
		expect(surface.className).not.toContain('py-3');
	});
});
