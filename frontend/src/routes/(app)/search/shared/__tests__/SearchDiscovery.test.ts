// SearchDiscovery 契约测试：发现态渲染与选择/清除回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SearchDiscovery from '../SearchDiscovery.svelte';

describe('SearchDiscovery', () => {
	it('渲染最近搜索/热门/话题区域', () => {
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], hot: ['云龙湖'], topics: ['摄影'], onChoose: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('最近搜索')).toBeTruthy();
		expect(screen.getByText('正在关注')).toBeTruthy();
		expect(screen.getByText('推荐话题')).toBeTruthy();
		expect(screen.getByText('回龙窝')).toBeTruthy();
	});

	it('空最近搜索显示提示文案', () => {
		render(SearchDiscovery, {
			props: { recent: [], hot: [], topics: [], onChoose: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('还没有搜索记录。')).toBeTruthy();
	});

	it('点击最近搜索项触发 onChoose、清除触发 onClear', async () => {
		const onChoose = vi.fn();
		const onClear = vi.fn();
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], hot: [], topics: [], onChoose, onClear }
		});
		await fireEvent.click(screen.getByRole('button', { name: '重新搜索 回龙窝' }));
		expect(onChoose).toHaveBeenCalledWith('回龙窝');
		await fireEvent.click(screen.getByRole('button', { name: '清除' }));
		expect(onClear).toHaveBeenCalledTimes(1);
	});
});
