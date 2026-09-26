// SectionMoments 墓碑测试：已删除瞬间展示占位而非空白卡（契约：客户端展示占位）。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';

const mockPosts = vi.hoisted(() => ({
	posts: [
		{ id: 'm1', title: '活的瞬间', img: '', likes: 0, comments: 0, created_at: '', is_deleted: false },
		{ id: 'm2', title: '', img: '', likes: 0, comments: 0, created_at: '', is_deleted: true }
	]
}));

vi.mock('@/lib/profile-data.svelte', () => ({
	useAccountData: () => ({ nickname: '我', avatar: '', locationName: '' }),
	useGalleryMoments: () => ({ posts: mockPosts.posts, loading: false, error: null, needsAuth: false, retry: vi.fn() }),
	useReactedMoments: () => ({ posts: [], loading: false, error: null, needsAuth: false, retry: vi.fn() }),
	useFavoriteMoments: () => ({ posts: [], loading: false, error: null, needsAuth: false, retry: vi.fn() })
}));

vi.mock('@/components/moments/moment-feed.svelte', () => ({
	createFeedInteractions: (get: () => unknown[]) => ({
		items: get(),
		toggleLike: vi.fn(),
		toggleSave: vi.fn()
	})
}));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({ page: { url: new URL('http://x/me/moments') } }));
vi.mock('@/stores/composer.svelte', () => ({ openComposer: vi.fn() }));

import SectionMoments from '../SectionMoments.svelte';

describe('SectionMoments tombstone', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('正常卡渲染内容，已删卡渲染占位', () => {
		render(SectionMoments);
		expect(screen.getByText('活的瞬间')).toBeTruthy();
		expect(screen.getByText('该瞬间已删除')).toBeTruthy();
	});

	it('墓碑不可点：无指向已删瞬间的详情入口', () => {
		render(SectionMoments);
		const tombstone = screen.getByText('该瞬间已删除').closest('div');
		expect(tombstone?.querySelector('a,button')).toBeNull();
	});
});
