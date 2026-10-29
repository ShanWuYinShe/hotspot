// MomentPostBody 详情改版契约（2026-10-29）：
// 单图按原比例完整展示（object-contain、无固定 cover 高度）；动作条所有端内嵌；评论区可注入输入器。
import { describe, it, expect, vi } from 'vitest';
import { createRawSnippet } from 'svelte';
import { render, screen } from '@testing-library/svelte';
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
	it('单图按原始比例完整展示：object-contain，不再用固定 cover 高度裁切', () => {
		const { container } = render(MomentPostBody, { props: baseProps });
		const img = container.querySelector('article img') as HTMLImageElement;
		expect(img).toBeTruthy();
		expect(img.className).toContain('object-contain');
		expect(img.className).not.toContain('object-cover');
		expect(img.className).not.toContain('h-cover-height-lg');
		expect(img.className).toContain('max-h-');
	});

	it('动作条在所有端内嵌（移动端不再依赖底部独立动作条）', () => {
		render(MomentPostBody, { props: baseProps });
		expect(screen.getByRole('button', { name: '点赞' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '查看评论' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '收藏' })).toBeTruthy();
	});

	it('commentComposer snippet 注入评论区顶部', () => {
		const composer = createRawSnippet(() => ({ render: () => '<div data-test="composer">输入器</div>' }));
		render(MomentPostBody, { props: { ...baseProps, commentComposer: composer } });
		const node = document.querySelector('[data-test="composer"]');
		expect(node).toBeTruthy();
		expect(node?.closest('section')?.id).toBe('moment-comments');
	});
});
