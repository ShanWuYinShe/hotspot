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

	it('priority 透传首屏图片优先级（eager + fetchpriority=high），默认 lazy', () => {
		// R17 首屏图策略：列表首屏前 2 张 eager+high，其余 lazy。
		const { container, unmount } = render(GroupCard, {
			props: { group, onToggle: () => {}, onOpen: () => {}, priority: true }
		});
		const hot = container.querySelector('img') as HTMLImageElement;
		expect(hot.getAttribute('loading')).toBe('eager');
		expect(hot.getAttribute('fetchpriority')).toBe('high');
		unmount();

		const def = render(GroupCard, { props: { group, onToggle: () => {}, onOpen: () => {} } });
		const cold = def.container.querySelector('img') as HTMLImageElement;
		expect(cold.getAttribute('loading')).toBe('lazy');
	});

	it('featured 变体渲染精选徽章与扩展信息', () => {
		const featured = {
			id: 'g2',
			name: '回龙窝胶片会',
			desc: '城市漫游与胶片记录',
			banner: 'https://example.com/b.jpg',
			category: '摄影',
			privacy: 'public',
			members: 143,
			online: 12,
			distance: '0.4km',
			activity: '周六外拍集合',
			joined: false,
			avatars: []
		} as never;
		render(GroupCard, { props: { group: featured, onToggle: () => {}, onOpen: () => {}, featured: true } });
		expect(screen.getByText('本周精选')).toBeTruthy();
		expect(screen.getByText('回龙窝胶片会')).toBeTruthy();
		expect(screen.getByText('城市漫游与胶片记录')).toBeTruthy();
		expect(screen.getByText('12 在线')).toBeTruthy();
	});
});
