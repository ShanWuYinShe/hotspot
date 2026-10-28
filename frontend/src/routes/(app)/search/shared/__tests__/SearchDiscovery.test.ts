// SearchDiscovery 契约测试：发现态渲染与选择/清除回调。
// 热门话题（moments/search 热门模式 tags 聚合）点击走 onChooseTopic（tag 语义搜索），
// 最近搜索点击走 onChoose（关键词搜索），热门动态点击走 onOpenMoment（详情导航），入口分流。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SearchDiscovery from '../SearchDiscovery.svelte';

describe('SearchDiscovery', () => {
	it('渲染最近搜索区域', () => {
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], onChoose: () => {}, onChooseTopic: () => {}, onOpenMoment: () => {}, onOpenUser: () => {}, onSeeAllUsers: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('最近搜索')).toBeTruthy();
		expect(screen.getByText('回龙窝')).toBeTruthy();
	});

	it('空最近搜索隐藏整个区块（r11：干瘪空态文案无信息量）', () => {
		render(SearchDiscovery, {
			props: { recent: [], onChoose: () => {}, onChooseTopic: () => {}, onOpenMoment: () => {}, onOpenUser: () => {}, onSeeAllUsers: () => {}, onClear: () => {} }
		});
		expect(screen.queryByText('最近搜索')).toBeNull();
		expect(screen.queryByLabelText('最近搜索')).toBeNull();
	});

	it('点击最近搜索项触发 onChoose、清除触发 onClear', async () => {
		const onChoose = vi.fn();
		const onClear = vi.fn();
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], onChoose, onChooseTopic: () => {}, onOpenMoment: () => {}, onOpenUser: () => {}, onSeeAllUsers: () => {}, onClear }
		});
		await fireEvent.click(screen.getByRole('button', { name: '重新搜索 回龙窝' }));
		expect(onChoose).toHaveBeenCalledWith('回龙窝');
		await fireEvent.click(screen.getByRole('button', { name: '清除' }));
		expect(onClear).toHaveBeenCalledTimes(1);
	});

	it('渲染热门话题并触发 onChooseTopic（tag 语义搜索分流）', async () => {
		const onChooseTopic = vi.fn();
		render(SearchDiscovery, {
			props: { recent: [], hotTopics: ['松弛'], onChoose: () => {}, onChooseTopic, onOpenMoment: () => {}, onOpenUser: () => {}, onSeeAllUsers: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('热门话题')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '#松弛' }));
		expect(onChooseTopic).toHaveBeenCalledWith('松弛');
	});

	it('热门动态区块：复用发现态数据渲染行，点击走 onOpenMoment（详情导航）', async () => {
		const onOpenMoment = vi.fn();
		render(SearchDiscovery, {
			props: {
				recent: [],
				hotMoments: [
					{ moment_id: 'm1', description: '周末市集', created_at: new Date().toISOString() } as never
				],
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment,
				onOpenUser: () => {},
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		expect(screen.getByText('热门动态')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '周末市集' }));
		expect(onOpenMoment).toHaveBeenCalledWith('m1');
	});

	it('人物推荐区块：渲染行并整行点击走 onOpenUser（资料页导航）', async () => {
		const onOpenUser = vi.fn();
		render(SearchDiscovery, {
			props: {
				recent: [],
				hotUsers: [
					{ id: 'u1', name: '林小满', avatar: '', tag: '摄影', distance: '1.2km', isOnline: true }
				],
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser,
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		expect(screen.getByText('人物推荐')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '查看 林小满 的资料' }));
		expect(onOpenUser).toHaveBeenCalledWith('u1');
	});

	it('无推荐用户时人物推荐区块不渲染（诚实空态）', () => {
		render(SearchDiscovery, {
			props: {
				recent: [],
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser: () => {},
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		expect(screen.queryByText('人物推荐')).toBeNull();
		expect(screen.queryByLabelText('人物推荐')).toBeNull();
	});

	it('推荐群组区块：渲染行并点击走 onOpenGroup，无群组时不渲染', async () => {
		const onOpenGroup = vi.fn();
		const onSeeAllGroups = vi.fn();
		const { rerender } = render(SearchDiscovery, {
			props: {
				recent: [],
				hotGroups: [{ id: 'g1', name: '周末爬山社', avatar: '', category: '户外', memberCount: 28 }],
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser: () => {},
				onOpenGroup,
				onSeeAllUsers: () => {},
				onSeeAllGroups,
				onClear: () => {}
			}
		});
		expect(screen.getByText('推荐群组')).toBeTruthy();
		expect(screen.getByText('周末爬山社')).toBeTruthy();
		expect(screen.getByText('28 成员')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '查看 周末爬山社 群组' }));
		expect(onOpenGroup).toHaveBeenCalledWith('g1');

		await rerender({
			recent: [],
			hotGroups: [],
			onChoose: () => {},
			onChooseTopic: () => {},
			onOpenMoment: () => {},
			onOpenUser: () => {},
			onSeeAllUsers: () => {},
			onClear: () => {}
		});
		expect(screen.queryByText('推荐群组')).toBeNull();
	});

	// ── R17 CLS 整改：发现态加载期骨架（与最终内容等高，数据到达只替换块内内容）──
	it('加载期四个分区渲染等高骨架（role=status 播报，骨架不进焦点序列）', () => {
		const { container } = render(SearchDiscovery, {
			props: {
				recent: [],
				topicsLoading: true,
				momentsLoading: true,
				usersLoading: true,
				groupsLoading: true,
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser: () => {},
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		// 分区骨架常驻最终位置：三块列表/画廊骨架 + 话题 chips 骨架都在首帧渲染
		for (const label of ['加载热门话题…', '加载人物推荐…', '加载推荐群组…', '加载热门动态…']) {
			const status = screen.getByRole('status', { name: label });
			expect(status.getAttribute('aria-busy')).toBe('true');
		}
		// 人物推荐/推荐群组各 3 行（与 limit 3 一致）：合计 6 行
		expect(container.querySelectorAll('[data-skeleton-row]').length).toBe(6);
		// 热门动态 4 卡（与 slice(0, 4) 一致）
		expect(container.querySelectorAll('[data-skeleton-card]').length).toBe(4);
		// 热门话题 chip 骨架（与 slice(0, 10) 的典型 2 行估算一致）
		expect(container.querySelectorAll('[data-skeleton]').length).toBeGreaterThanOrEqual(6);
	});

	it('数据到达后同位置替换为真实内容，骨架不再残留', async () => {
		const { container, rerender } = render(SearchDiscovery, {
			props: {
				recent: [],
				topicsLoading: true,
				momentsLoading: true,
				usersLoading: true,
				groupsLoading: true,
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser: () => {},
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		expect(container.querySelectorAll('[data-skeleton-card]').length).toBe(4);
		await rerender({
			recent: [],
			hotTopics: ['松弛'],
			hotMoments: [{ moment_id: 'm1', description: '周末市集', created_at: new Date().toISOString() } as never],
			hotUsers: [{ id: 'u1', name: '鹿鸣', avatar: '', tag: '摄影', distance: '300m', isOnline: true }],
			hotGroups: [{ id: 'g1', name: '晨跑团', avatar: '', category: '户外', memberCount: 5 }],
			topicsLoading: false,
			momentsLoading: false,
			usersLoading: false,
			groupsLoading: false,
			onChoose: () => {},
			onChooseTopic: () => {},
			onOpenMoment: () => {},
			onOpenUser: () => {},
			onSeeAllUsers: () => {},
			onClear: () => {}
		});
		expect(container.querySelectorAll('[data-skeleton-card]').length).toBe(0);
		expect(container.querySelectorAll('[data-skeleton-row]').length).toBe(0);
		expect(screen.queryAllByRole('status')).toHaveLength(0);
		for (const text of ['热门话题', '人物推荐', '推荐群组', '热门动态']) {
			expect(screen.getByText(text)).toBeTruthy();
		}
	});

	it('加载落定后无数据不渲染死骨架（诚实空态）', () => {
		const { container } = render(SearchDiscovery, {
			props: {
				recent: [],
				topicsLoading: false,
				momentsLoading: false,
				usersLoading: false,
				groupsLoading: false,
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser: () => {},
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		expect(container.querySelectorAll('[data-skeleton-card]').length).toBe(0);
		expect(container.querySelectorAll('[data-skeleton-row]').length).toBe(0);
		expect(screen.queryAllByRole('status')).toHaveLength(0);
		expect(screen.queryByText('热门话题')).toBeNull();
		expect(screen.queryByText('热门动态')).toBeNull();
	});

	// 回归守卫：两个行内文字动作（清除/查看全部）实测仅 22.7px 高，低于 WCAG 2.5.8 的
	// 24px 下限；必须带 hit-slop 扩热区（与首页 SectionHead 同名钮写法一致）。
	it('行内动作「清除/查看全部」带 hit-slop 扩热区', () => {
		render(SearchDiscovery, {
			props: {
				recent: ['回龙窝'],
				hotUsers: [{ id: 'u1', name: '鹿鸣', avatar: '', tag: '摄影', distance: '300m', isOnline: true }],
				onChoose: () => {},
				onChooseTopic: () => {},
				onOpenMoment: () => {},
				onOpenUser: () => {},
				onSeeAllUsers: () => {},
				onClear: () => {}
			}
		});
		for (const name of ['清除', '查看全部']) {
			expect(screen.getByRole('button', { name }).className).toContain('hit-slop');
		}
	});
});