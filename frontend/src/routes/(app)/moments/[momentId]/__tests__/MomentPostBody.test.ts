// MomentPostBody 详情改版契约（2026-10-29）：
// 单图按原比例完整展示（object-contain、无固定 cover 高度）；动作条所有端内嵌；评论区可注入输入器。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MomentPostBody from '../MomentPostBody.svelte';

const moment = {
	id: 'moment_1',
	authorId: 'user_lumos',
	user: '鹿鸣',
	avatar: '',
	location: '植物园',
	images: ['https://images.example.com/a.jpg'],
	content: '清晨在绿意中找到了一片宁静',
	time: '9月14日 15:25',
	likes: 30,
	tag: '自然',
	comments: []
};

const baseProps = {
	moment,
	comments: [],
	commentCount: 0,
	hasMoreComments: false,
	loadingMoreComments: false,
	liked: false,
	saved: false,
	shareStatus: 'idle' as const,
	onToggleLike: () => {},
	onToggleSave: () => {},
	onShare: () => {},
	onCommentClick: () => {},
	onReply: () => {},
	onExpandReplies: () => {},
	onToggleCommentLike: () => {},
	onReportComment: () => {},
	onDeleteComment: () => {},
	onLoadMoreComments: () => {}
};

describe('MomentPostBody 详情媒体与布局契约', () => {
	it('单图画廊展陈：主图按原比例整宽展示，不裁切、不压缩高度', () => {
		const { container } = render(MomentPostBody, { props: baseProps });
		const img = container.querySelector('[data-gallery-image]') as HTMLImageElement;
		expect(img).toBeTruthy();
		expect(img.className).toContain('object-contain');
		expect(img.className).not.toContain('object-cover');
		expect(img.className).not.toContain('h-cover-height-lg');
		expect(img.className).toContain('w-full');
		expect(img.className).toContain('h-auto');
		// 2026-10-29 用户反馈：不得用 max-h 压高度（高图由页面滚动承载、左右不留白）
		expect(img.className).not.toContain('max-h-');
	});

	it('图片区只保留顶部圆角（底部直角贴边），不是悬浮的圆角矩形', () => {
		const { container } = render(MomentPostBody, { props: baseProps });
		const img = container.querySelector('[data-gallery-image]') as HTMLImageElement;
		const box = img.closest('.relative.overflow-hidden') as HTMLElement;
		expect(box).toBeTruthy();
		expect(box.className).toContain('rounded-t-lg');
		expect(box.className).not.toContain('rounded-lg');
	});

	it('图片加载失败兜底：同图模糊铺底（ambient matte）仍保留', () => {
		const { container } = render(MomentPostBody, { props: baseProps });
		const backdrop = container.querySelector('[data-gallery-backdrop]') as HTMLImageElement;
		expect(backdrop).toBeTruthy();
		expect(backdrop.getAttribute('aria-hidden')).toBe('true');
		expect(backdrop.className).toContain('object-cover');
		// 模糊半径走 token（inline style：filter:blur(var(--cst-blur-lg))），不写死像素值
		expect(backdrop.getAttribute('style')).toContain('blur(var(--cst-blur-lg))');
	});

	it('动作条在所有端内嵌（移动端不再依赖底部独立动作条）', () => {
		render(MomentPostBody, { props: baseProps });
		expect(screen.getByRole('button', { name: '点赞' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '查看评论' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '收藏' })).toBeTruthy();
	});

	it('多图画廊：切换缩略图时保留双层交叉过渡，角标同步', async () => {
		const multiImageMoment = {
			...moment,
			images: ['https://images.example.com/1.jpg', 'https://images.example.com/2.jpg']
		};
		const { container } = render(MomentPostBody, { props: { ...baseProps, moment: multiImageMoment } });
		const thumbs = screen.getAllByRole('button', { name: /第 \d 张图片/ });
		expect(thumbs).toHaveLength(2);
		expect(thumbs[0].getAttribute('aria-pressed')).toBe('true');
		expect(thumbs[1].getAttribute('aria-pressed')).toBe('false');

		// 初始角标 1 / 2
		const badge = container.querySelector('.media-badge');
		expect(badge?.textContent?.trim()).toBe('1 / 2');

		// 点击第二张缩略图
		await fireEvent.click(thumbs[1]);
		expect(thumbs[1].getAttribute('aria-pressed')).toBe('true');
		expect(badge?.textContent?.trim()).toBe('2 / 2');

		// 主图依旧存在，具备过渡与平滑类
		const currentImg = container.querySelector('[data-gallery-image]') as HTMLImageElement;
		expect(currentImg).toBeTruthy();
		expect(currentImg.className).toContain('duration-rapid');
	});
});
