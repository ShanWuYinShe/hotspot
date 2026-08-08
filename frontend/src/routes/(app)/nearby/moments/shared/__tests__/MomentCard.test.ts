// MomentCard 契约测试：动态卡信息渲染与点赞/收藏回调。
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

	it('点击点赞/收藏按钮触发回调', async () => {
		const onLike = vi.fn();
		const onSave = vi.fn();
		const { container } = render(MomentCard, {
			props: { item, liked: false, saved: false, onLike, onSave }
		});
		const buttons = container.querySelectorAll('button');
		await fireEvent.click(buttons[0]);
		await fireEvent.click(buttons[1]);
		expect(onLike).toHaveBeenCalledTimes(1);
		expect(onSave).toHaveBeenCalledTimes(1);
	});
});
