// SearchInput 契约测试：渲染、受控回调、清空（含回焦）、Esc、读屏名称与输入族样式契约。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SearchInput from '../SearchInput.svelte';

describe('SearchInput', () => {
	it('渲染输入框与 placeholder', () => {
		render(SearchInput, { props: { value: '', onChange: () => {}, placeholder: '搜索会话…' } });
		expect(screen.getByPlaceholderText('搜索会话…')).toBeTruthy();
	});

	it('输入触发 onChange 并传递新值', async () => {
		const onChange = vi.fn();
		render(SearchInput, { props: { value: '', onChange } });
		const input = screen.getByRole('textbox') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: '鹿鸣' } });
		expect(onChange).toHaveBeenCalledWith('鹿鸣');
	});

	it('默认读屏名称为「搜索」', () => {
		render(SearchInput, { props: { value: '', onChange: () => {} } });
		expect(screen.getByRole('textbox', { name: '搜索' })).toBeTruthy();
	});

	it('ariaLabel 覆盖读屏名称（不同页面按语义命名）', () => {
		render(SearchInput, { props: { value: '', onChange: () => {}, ariaLabel: '搜索会话' } });
		expect(screen.getByRole('textbox', { name: '搜索会话' })).toBeTruthy();
	});

	it('value 非空时显示清空按钮，点击触发 onChange("")', async () => {
		const onChange = vi.fn();
		render(SearchInput, { props: { value: '已有输入', onChange } });
		const clear = screen.getByLabelText('清空搜索');
		expect(clear).toBeTruthy();
		await fireEvent.click(clear);
		expect(onChange).toHaveBeenCalledWith('');
	});

	it('点击清空后焦点回到输入框（连续清空不用再点一次）', async () => {
		const onChange = vi.fn();
		render(SearchInput, { props: { value: '已有输入', onChange } });
		const input = screen.getByRole('textbox') as HTMLInputElement;
		await fireEvent.click(screen.getByLabelText('清空搜索'));
		expect(document.activeElement).toBe(input);
	});

	it('Escape 清空输入', async () => {
		const onChange = vi.fn();
		render(SearchInput, { props: { value: '已有输入', onChange } });
		await fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Escape' });
		expect(onChange).toHaveBeenCalledWith('');
	});

	it('value 为空时不显示清空按钮', () => {
		render(SearchInput, { props: { value: '', onChange: () => {} } });
		expect(screen.queryByLabelText('清空搜索')).toBeNull();
	});

	it('搜索表面契约：胶囊 + 39px 档（px-3.5 py-2）+ text-md', () => {
		const { container } = render(SearchInput, { props: { value: '', onChange: () => {} } });
		const surface = container.querySelector('.input-surface') as HTMLElement;
		expect(surface.className).toContain('rounded-full');
		expect(surface.className).toContain('px-3.5');
		expect(surface.className).toContain('py-2');
		expect((screen.getByRole('textbox') as HTMLElement).className).toContain('text-md');
	});

	it('清空按钮 32px 视觉 + hit-slop 命中扩展（44px 触控下限）', () => {
		render(SearchInput, { props: { value: 'x', onChange: () => {} } });
		const clear = screen.getByLabelText('清空搜索');
		expect(clear.className).toContain('hit-slop');
		expect(clear.className).toContain('size-8');
	});

	it('搜索输入带 search 语义属性（移动键盘搜索键 + 关闭自动更正）', () => {
		render(SearchInput, { props: { value: '', onChange: () => {} } });
		const input = screen.getByRole('textbox') as HTMLInputElement;
		expect(input.getAttribute('enterkeyhint')).toBe('search');
		expect(input.getAttribute('autocomplete')).toBe('off');
		expect(input.hasAttribute('data-search-input')).toBe(true);
	});
});
