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
	it('加载态显示搜索中', () => {
		render(SearchResults, { props: base({ loading: true }) });
		expect(screen.getByText('搜索中…')).toBeTruthy();
	});

	it('空态显示没有找到查询词', () => {
		render(SearchResults, { props: base({}) });
		expect(screen.getByText('没有找到“鹿”')).toBeTruthy();
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
});
