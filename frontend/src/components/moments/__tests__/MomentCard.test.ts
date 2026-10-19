// MomentCard 契约测试：动态卡信息渲染与点赞/评论/收藏回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MomentCard from '../MomentCard.svelte';

const item = {
	id: 'm1',
	tag: '城市',
	user: '鹿鸣',
	avatar: '',
	location: '回龙窝',
	content: '城市里那些安静的角落',
	image: '',
	images: [],
	likes: 12,
	comments: 3
} as never;

describe('MomentCard', () => {
	it('渲染标签/作者/正文/点赞评论数（卡片不渲染位置）', () => {
		render(MomentCard, {
			props: { item, liked: false, saved: false, onLike: () => {}, onSave: () => {} }
		});
		expect(screen.getByText('城市')).toBeTruthy();
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('城市里那些安静的角落')).toBeTruthy();
		expect(screen.getByText('12')).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
		// 2026-10-20 契约变更：手机两列卡宽 171px，位置+时间会把昵称挤成 1 个字，
		// 卡片只保留时间；位置由详情页承载。此处断言"位置不出现在卡片上"。
		expect(screen.queryByText('回龙窝')).toBeNull();
	});

	it('点击点赞/评论/收藏按钮触发回调', async () => {
		const onLike = vi.fn();
		const onSave = vi.fn();
		const onClick = vi.fn();
		render(MomentCard, {
			props: { item, liked: false, saved: false, onLike, onSave, onClick }
		});
		// 用 aria-label 定位（不依赖 DOM 顺序；评论计数也是可点按钮）
		await fireEvent.click(screen.getByRole('button', { name: '点赞' }));
		await fireEvent.click(screen.getByRole('button', { name: '查看评论' }));
		await fireEvent.click(screen.getByRole('button', { name: '收藏' }));
		expect(onLike).toHaveBeenCalledTimes(1);
		expect(onSave).toHaveBeenCalledTimes(1);
		// 评论计数点击进入动态详情（评论区在详情页内）
		expect(onClick).toHaveBeenCalledTimes(1);
	});
	// 回归守卫（全站复审 s3 P2-7）：时间此前只在「无位置」的 {:else if} 兜底分支里，
	// 而个人中心给 location 兜了账号位置 → 发布时间永不渲染。位置与时间必须同时可见。
	// 注：上面的 item 是 `as never`，不能 spread，这里显式构造。
	const cardItem = (extra: Record<string, unknown>) =>
		({
			id: 'm1',
			tag: '城市',
			user: '鹿鸣',
			avatar: '',
			location: '回龙窝',
			content: '城市里那些安静的角落',
			image: '',
			images: [],
			likes: 12,
			comments: 3,
			...extra
		}) as never;

	it('位置与时间同时存在时，卡片只渲染时间', () => {
		render(MomentCard, {
			props: { item: cardItem({ time: '9月3日' }), liked: false, saved: false, onLike: () => {}, onSave: () => {} }
		});
		expect(screen.getByText('9月3日')).toBeTruthy();
		expect(screen.queryByText('回龙窝')).toBeNull();
	});

	it('无位置时仍渲染时间（时间独立于位置）', () => {
		render(MomentCard, {
			props: { item: cardItem({ location: '', time: '9月3日' }), liked: false, saved: false, onLike: () => {}, onSave: () => {} }
		});
		expect(screen.getByText('9月3日')).toBeTruthy();
	});
});
