// ProfileCompleteness 契约测试：资料完成度卡的两种形态——
// <100%：百分比 + 分段点 + 建议补全入口；
// 100%：不画进度条（长条会被读成装饰横线），只给完成态。
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

import ProfileCompleteness from '../ProfileCompleteness.svelte';
import type { CompletenessItem } from '../ProfileCompleteness.svelte';

function item(id: string, done: boolean): CompletenessItem {
	return { id, label: `补全${id}`, done, href: '/me/edit' };
}

describe('ProfileCompleteness', () => {
	it('未完成：展示百分比、分段点与建议补全项', () => {
		render(ProfileCompleteness, { props: { items: [item('avatar', true), item('bio', true), item('group', false)] } });
		expect(screen.getByText('67%')).toBeTruthy();
		// 分段点容器带 progressbar 语义（进度不是纯装饰）
		const bar = screen.getByRole('progressbar', { name: '资料完整度' });
		expect(bar.getAttribute('aria-valuenow')).toBe('67');
		expect(screen.getByText('建议补全')).toBeTruthy();
		expect(screen.getByText('补全group')).toBeTruthy();
		expect(screen.queryByText('资料已完善')).toBeNull();
	});

	it('已完成：不渲染进度条与百分比，只给完成态说明', () => {
		render(ProfileCompleteness, { props: { items: [item('avatar', true), item('bio', true)] } });
		expect(screen.queryByRole('progressbar')).toBeNull();
		// 100% 时不再展示「100%」与长进度条（长条会被读成装饰横线）
		expect(screen.queryByText('100%')).toBeNull();
		expect(screen.getByText('资料已完善')).toBeTruthy();
		expect(screen.queryByText('建议补全')).toBeNull();
	});
});