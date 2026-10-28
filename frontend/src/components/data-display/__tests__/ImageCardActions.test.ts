// ImageCardActions 契约测试：计数展示、状态语义（点赞红/收藏 accent）、回调与 aria。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ImageCardActions from '../ImageCardActions.svelte';

const base = {
	likes: 10,
	comments: 3,
	shares: 2,
	liked: false,
	saved: false,
	onLike: () => {},
	onComment: () => {},
	onSave: () => {},
	onShare: () => {}
};

describe('ImageCardActions', () => {
	it('渲染点赞/评论/分享/收藏四类操作与计数', () => {
		render(ImageCardActions, { props: base });
		expect(screen.getByRole('button', { name: '点赞' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '查看评论' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '分享瞬间' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '收藏' })).toBeTruthy();
		expect(screen.getByText('10')).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
		expect(screen.getByText('2')).toBeTruthy();
	});

	it('点赞态：计数 +1、aria-pressed、危险红填充（压图态白描边）', () => {
		const { container } = render(ImageCardActions, { props: { ...base, liked: true } });
		expect(screen.getByText('11')).toBeTruthy();
		const likeBtn = screen.getByRole('button', { name: '取消点赞' });
		expect(likeBtn.getAttribute('aria-pressed')).toBe('true');
		const heart = container.querySelector('.lucide-heart');
		expect(heart?.getAttribute('class')).toContain('fill-status-danger');
		expect(heart?.getAttribute('class')).toContain('text-content-inverse');
	});

	it('收藏态：accent 填充；非压图态用主题色而非反色', () => {
		const { container } = render(ImageCardActions, { props: { ...base, saved: true, onImage: false } });
		const bookmark = container.querySelector('.lucide-bookmark');
		expect(bookmark?.getAttribute('class')).toContain('fill-accent-default');
		expect(bookmark?.getAttribute('class')).toContain('text-accent-default');
	});

	it('非压图（fallback/实底）态：点赞激活用 status-danger 文字而非反色', () => {
		render(ImageCardActions, { props: { ...base, liked: true, onImage: false } });
		const btn = screen.getByRole('button', { name: '取消点赞' });
		expect(btn.className).toContain('text-status-danger');
	});

	it('点击回调：点赞/收藏传下一状态，评论/分享无参数', async () => {
		const onLike = vi.fn();
		const onSave = vi.fn();
		const onComment = vi.fn();
		const onShare = vi.fn();
		render(ImageCardActions, { props: { ...base, onLike, onSave, onComment, onShare } });

		await fireEvent.click(screen.getByRole('button', { name: '点赞' }));
		expect(onLike).toHaveBeenCalledWith(true);
		await fireEvent.click(screen.getByRole('button', { name: '收藏' }));
		expect(onSave).toHaveBeenCalledWith(true);
		await fireEvent.click(screen.getByRole('button', { name: '查看评论' }));
		expect(onComment).toHaveBeenCalledTimes(1);
		await fireEvent.click(screen.getByRole('button', { name: '分享瞬间' }));
		expect(onShare).toHaveBeenCalledTimes(1);
	});

	it('计数为 0 且 placeholderLabels 时显示「赞/评论」文案', () => {
		render(ImageCardActions, { props: { ...base, likes: 0, comments: 0 } });
		expect(screen.getByText('赞')).toBeTruthy();
		expect(screen.getByText('评论')).toBeTruthy();
	});
});