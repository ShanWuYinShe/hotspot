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
		// 互动统计：mock 中已点赞（reacted_by_me）→ ImageCardActions 展示基数 +1
		expect(screen.getByText('13')).toBeTruthy();
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

	it('卡片根节点自成堆叠上下文（isolate）且按压走统一 cardPress', async () => {
		const { container } = render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {} } });
		const root = container.firstElementChild as HTMLElement;
		expect(root.className.split(/\s+/)).toContain('isolate');
		await fireEvent.pointerDown(root);
		expect(root.dataset.pressed).toBe('true');
		await fireEvent.pointerUp(root);
		expect(root.dataset.pressed).toBeUndefined();
	});

	it('行内操作按钮按下不触发整卡按压态（避免点按钮整卡抖一下）', async () => {
		const { container } = render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {} } });
		const root = container.firstElementChild as HTMLElement;
		const likeBtn = screen.getByRole('button', { name: '取消点赞' });
		await fireEvent.pointerDown(likeBtn);
		expect(root.dataset.pressed).toBeUndefined();
	});

	it('覆盖操作行统一走 ImageCardActions（四类操作齐备，评论回调生效）', async () => {
		const onComment = vi.fn();
		render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {}, onComment } });
		expect(screen.getByRole('button', { name: '取消点赞' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '查看评论' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '分享瞬间' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '收藏' })).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '查看评论' }));
		expect(onComment).toHaveBeenCalledTimes(1);
	});

	it('加载中占位用限时脉冲类（media-placeholder--pulse），不再用 Tailwind animate-pulse', () => {
		// R17：旧 animate-pulse 是 2s/infinite，慢图与首屏外 lazy 卡会让占位动画常驻；
		// 改用 .media-placeholder--pulse（3 次后转静态）。jsdom 不跑 CSS 动画，
		// 按本文件惯例以类串断言锁定契约。
		const { container } = render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {} } });
		const placeholder = container.querySelector('.media-placeholder--pulse');
		expect(placeholder).not.toBeNull();
		expect(placeholder?.className).not.toContain('animate-pulse');
		expect(placeholder?.getAttribute('aria-hidden')).toBe('true');
		// 占位底色仍是主题 accent 渐变（与中性灰骨架屏区分）
		expect(placeholder?.className).toContain('surface-accent-gradient');
	});

	it('压图角标统一用 media-badge（分类/位置/已关注）', () => {
		const { container } = render(HomeFeedItem, { props: { moment: mockMoment, onClick: () => {} } });
		const badges = [...container.querySelectorAll('.media-badge')];
		expect(badges.length).toBeGreaterThanOrEqual(2);
	});
});
	// 2026-10-29 用户反馈回归：卡片上的「已关注」曾是不可点徽章，点击穿透整卡覆盖层打开详情。
	// 现在必须是真按钮：可取消关注、aria-pressed/label 正确、点击绝不触发整卡 onClick。
	it('已关注 chip 是可点按钮（aria-pressed + 取消关注 label），点击不触发整卡导航', async () => {
		const onClick = vi.fn();
		render(HomeFeedItem, { props: { moment: mockMoment, onClick } });
		const chip = screen.getByRole('button', { name: /取消关注 咖啡小迷妹/ }) as HTMLButtonElement;
		expect(chip.getAttribute('aria-pressed')).toBe('true');
		await fireEvent.click(chip);
		expect(onClick).not.toHaveBeenCalled();
	});

	it('未关注作者默认不显示关注 chip（避免信息流每张卡顶一颗按钮）', () => {
		render(HomeFeedItem, {
			props: { moment: { ...mockMoment, author_relation: { is_following: false, is_followed_by: false } }, onClick: () => {} }
		});
		expect(screen.queryByRole('button', { name: /关注 咖啡小迷妹/ })).toBeNull();
	});
