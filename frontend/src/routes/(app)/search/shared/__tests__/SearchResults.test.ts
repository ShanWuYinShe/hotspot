// SearchResults 契约测试：加载/空态/结果渲染与 Tab 切换。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SearchResults from '../SearchResults.svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

const tabs = [
	{ id: 'all', label: '全部' },
	{ id: 'people', label: '用户' },
	{ id: 'groups', label: '群组' }
] as const;

const base = (over: Record<string, unknown> = {}) => ({
	query: '鹿',
	tab: 'all',
	tabs,
	loading: false,
	hasResults: false,
	results: { users: [], moments: [], topics: [], groups: [] },
	onTabChange: () => {},
	onChoose: () => {},
	...over
}) as never;

describe('SearchResults', () => {
	it('加载态渲染搜索骨架屏（role=status 播报「搜索中…」，不再用 spinner 文案）', () => {
		const { container } = render(SearchResults, { props: base({ loading: true }) });
		const status = screen.getByRole('status');
		expect(status.getAttribute('aria-label')).toBe('搜索中…');
		expect(status.getAttribute('aria-busy')).toBe('true');
		expect(container.querySelectorAll('[data-skeleton-row]').length).toBeGreaterThan(0);
	});

	it('空态显示没有找到查询词', () => {
		render(SearchResults, { props: base({}) });
		expect(screen.getByText('没有找到“鹿”')).toBeTruthy();
	});

	it('单 tab 为空同样给空态（此前切到空 tab 一片空白）', () => {
		render(
			SearchResults,
			base({
				tab: 'people',
				hasResults: true,
				results: {
					users: [],
					moments: [{ moment_id: 'm1', description: '有内容' }],
					topics: [],
					groups: []
				}
			})
		);
		expect(screen.getByText('没有找到“鹿”')).toBeTruthy();
		expect(screen.queryByText('有内容')).toBeNull();
	});

	it('空态给建议 chips：点击走 onChooseTopic（tag 语义搜索）', async () => {
		const onChooseTopic = vi.fn();
		render(
			SearchResults,
			base({ hotTopics: ['松弛', '老街'], onChooseTopic })
		);
		expect(screen.getByText('没有找到“鹿”')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '#松弛' }));
		expect(onChooseTopic).toHaveBeenCalledWith('松弛');
	});

	it('渲染用户结果与 Tab 切换回调', async () => {
		const onTabChange = vi.fn();
		render(
			SearchResults,
			base({
				hasResults: true,
				onTabChange,
				results: {
					users: [{ user_id: 'u1', nickname: '鹿鸣' }],
					moments: [],
					topics: [],
					groups: []
				}
			})
		);
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '用户' }));
		expect(onTabChange).toHaveBeenCalledWith('people');
	});

	it('渲染群组结果组', () => {
		render(
			SearchResults,
			base({
				hasResults: true,
				results: {
					users: [],
					moments: [],
					topics: [],
					groups: [{ group_id: 'g1', name: '晨跑团', avatar_url: '', members_count: 5 }]
				}
			})
		);
		expect(screen.getByText('晨跑团')).toBeTruthy();
		expect(screen.getByText('5 位成员')).toBeTruthy();
		expect(screen.getAllByText('群组').length).toBeGreaterThan(0);
	});

	// 回归（审计 shell-home F2）：隐式 auto 轨会被行内 min-content 撑到 436px，正文列只有 358px
	// → 结果岛溢出 390 视口并被 main 的 overflow-x-hidden 裁剪。显式单列 = minmax(0,1fr)。
	it('结果栅格显式单列且分组 min-w-0（390 横向溢出回归）', () => {
		const { container } = render(
			SearchResults,
			base({
				hasResults: true,
				results: {
					users: [],
					moments: [{ moment_id: 'm1', description: '云龙湖的落日把水面染成金色' }],
					topics: [],
					groups: []
				}
			})
		);
		const grid = container.querySelector('[class*="lg:grid-cols-2"]') as HTMLElement;
		expect(grid).toBeTruthy();
		expect(grid.className).toContain('grid-cols-1');
		const group = grid.querySelector('section') as HTMLElement;
		expect(group.className.split(/\s+/)).toContain('min-w-0');
	});
});
