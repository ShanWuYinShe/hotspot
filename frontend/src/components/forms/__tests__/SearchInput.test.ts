// SearchInput 契约测试：输入回调、清空按钮显隐与清空回调。
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

	it('value 非空时显示清空按钮，点击触发 onChange("")', async () => {
		const onChange = vi.fn();
		render(SearchInput, { props: { value: '已有输入', onChange } });
		const clear = screen.getByLabelText('清空搜索');
		expect(clear).toBeTruthy();
		await fireEvent.click(clear);
		expect(onChange).toHaveBeenCalledWith('');
	});

	it('value 为空时不显示清空按钮', () => {
		render(SearchInput, { props: { value: '', onChange: () => {} } });
		expect(screen.queryByLabelText('清空搜索')).toBeNull();
	});
});
