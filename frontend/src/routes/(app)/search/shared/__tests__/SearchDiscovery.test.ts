// SearchDiscovery 契约测试：发现态渲染与选择/清除回调。
// 热门话题（moments/search 热门模式 tags 聚合）点击走 onChooseTopic（tag 语义搜索），
// 最近搜索点击走 onChoose（关键词搜索），热门动态点击走 onOpenMoment（详情导航），入口分流。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SearchDiscovery from '../SearchDiscovery.svelte';

describe('SearchDiscovery', () => {
	it('渲染最近搜索区域', () => {
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], onChoose: () => {}, onChooseTopic: () => {}, onOpenMoment: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('最近搜索')).toBeTruthy();
		expect(screen.getByText('回龙窝')).toBeTruthy();
	});

	it('空最近搜索隐藏整个区块（r11：干瘪空态文案无信息量）', () => {
		render(SearchDiscovery, {
			props: { recent: [], onChoose: () => {}, onChooseTopic: () => {}, onOpenMoment: () => {}, onClear: () => {} }
		});
		expect(screen.queryByText('最近搜索')).toBeNull();
		expect(screen.queryByLabelText('最近搜索')).toBeNull();
	});

	it('点击最近搜索项触发 onChoose、清除触发 onClear', async () => {
		const onChoose = vi.fn();
		const onClear = vi.fn();
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], onChoose, onChooseTopic: () => {}, onOpenMoment: () => {}, onClear }
		});
		await fireEvent.click(screen.getByRole('button', { name: '重新搜索 回龙窝' }));
		expect(onChoose).toHaveBeenCalledWith('回龙窝');
		await fireEvent.click(screen.getByRole('button', { name: '清除' }));
		expect(onClear).toHaveBeenCalledTimes(1);
	});

	it('渲染热门话题并触发 onChooseTopic（tag 语义搜索分流）', async () => {
		const onChooseTopic = vi.fn();
		render(SearchDiscovery, {
			props: { recent: [], hotTopics: ['松弛'], onChoose: () => {}, onChooseTopic, onOpenMoment: () => {}, onClear: () => {} }
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
				onClear: () => {}
			}
		});
		expect(screen.getByText('热门动态')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '周末市集' }));
		expect(onOpenMoment).toHaveBeenCalledWith('m1');
	});
});
