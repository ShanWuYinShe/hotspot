import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import NearbyHubSections from '../NearbyHubSections.svelte';
import type { MomentItem } from '@/api/types';
import type { NearbyUser } from '@/components/home/nearby-users';

// 锁定 2026-10-19 IA 调整后的「附近」枢纽契约：人 / 瞬间 / 群组 三段预览 + 查看全部入口，
// 空数据段不渲染（防止又退化回空框，或把某一段漏掉）。

const users: NearbyUser[] = Array.from({ length: 6 }, (_, i) => ({
	id: `user_${i}`,
	name: `用户${i}`,
	distance: '500m',
	avatar: '',
	isOnline: i % 2 === 0,
	tag: '户外',
	isFollowing: false
}));

// ActivityRow 渲染的是 description（见组件内 <p>{moment.description}</p>）
const moments: MomentItem[] = Array.from({ length: 5 }, (_, i) => ({
	moment_id: `moment_${i}`,
	content: `动态${i}`,
	description: `动态${i}`,
	created_at: '2026-10-19T00:00:00Z',
	images: [],
	tags: []
})) as unknown as MomentItem[];

const groups = [
	{ id: 'group_1', name: '爬山群', members: 12, banner: '', distance: '1km', privacy: 'public' as const }
];

function baseProps(overrides: Record<string, unknown> = {}) {
	return {
		users,
		moments,
		groups,
		userPreview: 4,
		momentPreview: 3,
		isFollowed: () => false,
		isPending: () => false,
		onToggleFollow: vi.fn(),
		onSeeAllUsers: vi.fn(),
		onSeeAllMoments: vi.fn(),
		onSeeAllGroups: vi.fn(),
		onOpenMoment: vi.fn(),
		...overrides
	};
}

describe('NearbyHubSections（附近枢纽三段分区）', () => {
	it('渲染三段，且各段只展示预览条数', () => {
		render(NearbyHubSections, { props: baseProps() });
		expect(screen.getByText('附近的人')).toBeTruthy();
		expect(screen.getByText('附近瞬间')).toBeTruthy();
		expect(screen.getByText('附近群组')).toBeTruthy();
		// 预览条数：人 4（共 6）、瞬间 3（共 5）
		expect(screen.getByText('用户3')).toBeTruthy();
		expect(screen.queryByText('用户4')).toBeNull();
		expect(screen.getByText('动态2')).toBeTruthy();
		expect(screen.queryByText('动态3')).toBeNull();
	});

	it('「查看全部」回调分别指向三个叶子页入口', async () => {
		const props = baseProps();
		render(NearbyHubSections, { props });
		// R13：三段卡头统一 SectionHead（人 / 瞬间 / 群组都由「查看全部」入口，不再有「更多」变体）
		const seeAll = screen.getAllByText('查看全部');
		expect(seeAll.length).toBe(3);
		await seeAll[0].click();
		await seeAll[1].click();
		await seeAll[2].click();
		expect(props.onSeeAllUsers).toHaveBeenCalledOnce();
		expect(props.onSeeAllMoments).toHaveBeenCalledOnce();
		expect(props.onSeeAllGroups).toHaveBeenCalledOnce();
	});

	it('空数据段整段不渲染', () => {
		const { container } = render(NearbyHubSections, {
			props: baseProps({ users: [], moments: [] })
		});
		expect(container.querySelector('[data-hub-section="people"]')).toBeNull();
		expect(container.querySelector('[data-hub-section="moments"]')).toBeNull();
		// 群组段仍在（仅在 groups 非空时渲染）
		expect(container.querySelector('[data-hub-section="groups"]')).not.toBeNull();
	});
});
