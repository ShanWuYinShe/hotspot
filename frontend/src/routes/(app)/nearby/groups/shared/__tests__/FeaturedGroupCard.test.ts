// FeaturedGroupCard 契约测试：精选卡信息渲染与加入切换。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import FeaturedGroupCard from '../FeaturedGroupCard.svelte';

const group = {
	id: 'g1',
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

describe('FeaturedGroupCard', () => {
	it('渲染精选徽章与群组信息', () => {
		render(FeaturedGroupCard, { props: { group, onToggle: () => {}, onOpen: () => {} } });
		expect(screen.getByText('本周精选')).toBeTruthy();
		expect(screen.getByText('回龙窝胶片会')).toBeTruthy();
		expect(screen.getByText('城市漫游与胶片记录')).toBeTruthy();
		expect(screen.getByText('143')).toBeTruthy();
		expect(screen.getByText('12 在线')).toBeTruthy();
		expect(screen.getByText('周六外拍集合')).toBeTruthy();
	});

	it('点击加入按钮触发 onToggle', async () => {
		const onToggle = vi.fn();
		render(FeaturedGroupCard, { props: { group, onToggle, onOpen: () => {} } });
		await fireEvent.click(screen.getByRole('button', { name: '加入' }));
		expect(onToggle).toHaveBeenCalledTimes(1);
	});
});
