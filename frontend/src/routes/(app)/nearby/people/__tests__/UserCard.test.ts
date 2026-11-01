// UserCard 无头像/头像加载失败时渲染主题化占位，而非浏览器 broken image。
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import UserCard, { type NearbyUser } from '@/components/data-display/UserCard.svelte';

const base = (over: Partial<NearbyUser> = {}): NearbyUser => ({
	id: 'user_1',
	name: '鹿鸣',
	distance: '0.3km',
	avatar: '',
	isOnline: true,
	tag: '摄影',
	tags: ['摄影', '旅行'],
	mutual: 2,
	note: '记录城市光影',
	isFollowing: false,
	...over
});

describe('UserCard', () => {
	it('avatar 为空时渲染首字母占位，不渲染 img', () => {
		const { container } = render(UserCard, {
			props: { user: base(), followed: false, onFollow: () => {}, onNavigate: () => {} }
		});
		expect(container.querySelector('img')).toBeNull();
		expect(screen.getByText('鹿')).toBeTruthy();
	});

	it('有头像时渲染 img；加载失败后切换为占位', async () => {
		const { container } = render(UserCard, {
			props: {
				user: base({ avatar: 'https://example.com/a.jpg' }),
				followed: false,
				onFollow: () => {},
				onNavigate: () => {}
			}
		});
		const img = container.querySelector('img');
		expect(img).not.toBeNull();
		await fireEvent.error(img as Element);
		expect(container.querySelector('img')).toBeNull();
		expect(screen.getByText('鹿')).toBeTruthy();
	});

	it('在线状态角标放置于 overflow-hidden 内层容器外部，避免被裁剪', () => {
		const { container } = render(UserCard, {
			props: {
				user: base({ isOnline: true }),
				followed: false,
				onFollow: () => {},
				onNavigate: () => {}
			}
		});
		const dot = container.querySelector('[title="在线"]');
		expect(dot).not.toBeNull();
		// 角标直接父级不包含 overflow-hidden
		const parent = dot?.parentElement;
		expect(parent?.classList.contains('overflow-hidden')).toBe(false);
	});
});
