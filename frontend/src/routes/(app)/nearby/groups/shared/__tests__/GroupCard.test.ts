// GroupCard 契约测试：群组信息渲染与加入切换回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import GroupCard from '../GroupCard.svelte';

const group = {
	id: 'g1',
	name: '云龙湖晨跑团',
	banner: 'https://example.com/b.jpg',
	category: '运动',
	privacy: 'public',
	members: 89,
	distance: '0.8km',
	joined: false,
	avatars: []
} as never;

describe('GroupCard', () => {
	it('渲染群组名称/分类/成员/距离', () => {
		render(GroupCard, { props: { group, onToggle: () => {}, onOpen: () => {} } });
		expect(screen.getByText('云龙湖晨跑团')).toBeTruthy();
		expect(screen.getByText('运动')).toBeTruthy();
		expect(screen.getByText('89 人')).toBeTruthy();
		expect(screen.getByText('0.8km')).toBeTruthy();
	});

	it('点击加入按钮触发 onToggle', async () => {
		const onToggle = vi.fn();
		render(GroupCard, { props: { group, onToggle, onOpen: () => {} } });
		await fireEvent.click(screen.getByRole('button', { name: '加入' }));
		expect(onToggle).toHaveBeenCalledTimes(1);
	});
});
