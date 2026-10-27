import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import HomeFeedItem from '../HomeFeedItem.svelte';
import type { MomentItem } from '@/api/types';

const mockMoment: MomentItem = {
	moment_id: 'moment_123',
	description: '在老街手冲馆度过了一个悠闲的午后，咖啡香气扑鼻。',
	images: [
		{
			url: 'https://images.example.com/coffee.jpg',
			thumbnail_url: null,
			width: 800,
			height: 600,
			mime_type: 'image/jpeg'
		}
	],
	tags: ['咖啡时光', '慢生活'],
	mentions: [],
	created_at: new Date(Date.now() - 3600_000).toISOString(),
	comments_count: 5,
	shares_count: 2,
	reactions: [
		{ emoji: '❤️', count: 12, reacted_by_me: true }
	],
	user: {
		user_id: 'user_456',
		nickname: '咖啡小迷妹',
		avatar_url: 'https://images.example.com/avatar.jpg'
	},
	location: {
		point: { latitude: 34.2, longitude: 117.2 },
		location_name: '老街手冲馆',
		distance: 450,
		bearing_degrees: null
	},
	is_favorited: false,
	author_relation: { is_following: true, is_followed_by: false },
	recommendation: 'trending',
	is_deleted: false
};

describe('HomeFeedItem', () => {
	it('完整渲染作者昵称、正文、配图、标签与位置信息', () => {
		render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {} } });

		// 作者昵称
		expect(screen.getByText('咖啡小迷妹')).toBeTruthy();
		// 关注状态与推荐标签
		expect(screen.getByText('已关注')).toBeTruthy();
		expect(screen.getByText('热门')).toBeTruthy();
		// 动态正文
		expect(screen.getByText('在老街手冲馆度过了一个悠闲的午后，咖啡香气扑鼻。')).toBeTruthy();
		// 位置信息
		expect(screen.getByText('老街手冲馆')).toBeTruthy();
		// 标签：整卡图片卡底部元信息只展示首个标签（Round 2 精简信息层级；全部标签见详情页）
		expect(screen.getByText('#咖啡时光')).toBeTruthy();
		// 互动统计
		expect(screen.getByText('12')).toBeTruthy();
		expect(screen.getByText('5')).toBeTruthy();
		expect(screen.getByText('2')).toBeTruthy();
	});

	it('R13 卡片语言：标题升档、时间不重复、hover 由 card-hover 原语承载', () => {
		const { container } = render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {} } });
		const title = screen.getByText('在老街手冲馆度过了一个悠闲的午后，咖啡香气扑鼻。');
		// 标题档：text-base sm:text-lg semibold leading-tight（压图白字、遮罩不变）
		expect(title.className).toContain('text-base');
		expect(title.className).toContain('sm:text-lg');
		expect(title.className).toContain('leading-tight');
		// 时间只在作者行出现一次（此前作者行 + 底部元信息重复渲染同一时间）
		expect(screen.getAllByText('1小时前').length).toBe(1);
		// 整卡 hover 走 card-hover（主题化上浮 + 图片 1.045 缩放），不再手写 hover 样式
		const card = container.firstElementChild as HTMLElement;
		expect(card.className).toContain('card-hover');
	});

	it('点击行触发 onClick 回调', async () => {
		const onClick = vi.fn();
		render(HomeFeedItem, { props: { moment: mockMoment, onClick } });

		const article = screen.getByRole('button', { name: /查看咖啡小迷妹的瞬间详情/i });
		await fireEvent.click(article);
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});
