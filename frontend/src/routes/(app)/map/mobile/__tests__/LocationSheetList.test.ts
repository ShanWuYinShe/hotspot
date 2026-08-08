// LocationSheetList 契约测试：搜索结果行渲染、选择回调与空态。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import LocationSheetList from '../LocationSheetList.svelte';

const base = {
	searchQuery: '回',
	searchResults: [{ displayName: '回龙窝' } as never],
	recentItems: [],
	onSelectLoc: () => {},
	onOpenPinDetail: () => {},
	onClearPin: () => {}
};

describe('LocationSheetList', () => {
	it('渲染搜索结果行并点击触发选择回调', async () => {
		const onSelectLoc = vi.fn();
		render(LocationSheetList, { props: { ...base, onSelectLoc } });
		expect(screen.getByText('回龙窝')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '选择地点 回龙窝' }));
		expect(onSelectLoc).toHaveBeenCalledTimes(1);
	});

	it('无匹配结果显示空态文案', () => {
		render(LocationSheetList, { props: { ...base, searchResults: [] } });
		expect(screen.getByText('未找到匹配地点')).toBeTruthy();
	});
});
