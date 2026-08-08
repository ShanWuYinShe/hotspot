// UserCardRow 契约测试：名称/头像渲染、可点击行与卡片形态。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import UserCardRow from '../UserCardRow.svelte';

describe('UserCardRow', () => {
	it('渲染名称与头像 alt', () => {
		render(UserCardRow, {
			props: { name: '鹿鸣', avatar: 'https://example.com/a.jpg' }
		});
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByAltText('鹿鸣')).toBeTruthy();
	});

	it('onClick 与 ariaLabel 齐备时可点击并触发回调', async () => {
		const onClick = vi.fn();
		render(UserCardRow, {
			props: { name: '鹿鸣', avatar: 'x', onClick, ariaLabel: '查看鹿鸣' }
		});
		await fireEvent.click(screen.getByRole('button', { name: '查看鹿鸣' }));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('tone=card 应用独立卡片行类', () => {
		const { container } = render(UserCardRow, {
			props: { name: '鹿鸣', avatar: 'x', tone: 'card', ariaLabel: '行' }
		});
		const row = container.querySelector('[aria-label="行"]') as HTMLElement;
		expect(row.className).toContain('rounded-md');
	});
});
