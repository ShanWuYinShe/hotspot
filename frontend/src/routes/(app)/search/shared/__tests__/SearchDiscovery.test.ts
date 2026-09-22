// SearchDiscovery 契约测试：发现态渲染与选择/清除回调。
// 热门/话题区块已移除（无服务端数据源，见组件注释），发现态只保留最近搜索。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SearchDiscovery from '../SearchDiscovery.svelte';

describe('SearchDiscovery', () => {
	it('渲染最近搜索区域', () => {
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], onChoose: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('最近搜索')).toBeTruthy();
		expect(screen.getByText('回龙窝')).toBeTruthy();
	});

	it('空最近搜索显示提示文案', () => {
		render(SearchDiscovery, {
			props: { recent: [], onChoose: () => {}, onClear: () => {} }
		});
		expect(screen.getByText('暂无搜索记录，输入关键词开始探索。')).toBeTruthy();
	});

	it('点击最近搜索项触发 onChoose、清除触发 onClear', async () => {
		const onChoose = vi.fn();
		const onClear = vi.fn();
		render(SearchDiscovery, {
			props: { recent: ['回龙窝'], onChoose, onClear }
		});
		await fireEvent.click(screen.getByRole('button', { name: '重新搜索 回龙窝' }));
		expect(onChoose).toHaveBeenCalledWith('回龙窝');
		await fireEvent.click(screen.getByRole('button', { name: '清除' }));
		expect(onClear).toHaveBeenCalledTimes(1);
	});
});
