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
	it('渲染标签/作者/位置/正文/点赞评论数', () => {
		render(MomentCard, {
			props: { item, liked: false, saved: false, onLike: () => {}, onSave: () => {} }
		});
		expect(screen.getByText('城市')).toBeTruthy();
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('回龙窝')).toBeTruthy();
		expect(screen.getByText('城市里那些安静的角落')).toBeTruthy();
		expect(screen.getByText('12')).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
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
});
