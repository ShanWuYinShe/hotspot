// AdminModuleCard 契约测试（/admin 概览模块入口卡）：文案/角标/禁用态/点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { MapPinned } from '@lucide/svelte';
import AdminModuleCard from '../AdminModuleCard.svelte';

describe('AdminModuleCard', () => {
	it('渲染标题与说明', () => {
		render(AdminModuleCard, {
			props: { label: '地理统计', description: '全站分布与热门地点', icon: MapPinned, onOpen: () => {} }
		});
		expect(screen.getByText('地理统计')).toBeTruthy();
		expect(screen.getByText('全站分布与热门地点')).toBeTruthy();
	});

	it('有待办角标时渲染角标文案', () => {
		render(AdminModuleCard, {
			props: { label: '举报审核', description: '裁决用户举报', icon: MapPinned, badge: '待处理 2', onOpen: () => {} }
		});
		expect(screen.getByText('待处理 2')).toBeTruthy();
	});

	it('无权限时按钮禁用并明示「无访问权限」', () => {
		render(AdminModuleCard, {
			props: { label: '用户管理', description: '搜索与封禁', icon: MapPinned, allowed: false, onOpen: () => {} }
		});
		const btn = screen.getByRole('button') as HTMLButtonElement;
		expect(btn.disabled).toBe(true);
		expect(screen.getByText('无访问权限')).toBeTruthy();
	});

	it('点击触发 onOpen', async () => {
		const onOpen = vi.fn();
		render(AdminModuleCard, {
			props: { label: '系统配置', description: '应用配置', icon: MapPinned, onOpen }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onOpen).toHaveBeenCalledTimes(1);
	});
});
