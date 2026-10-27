// IdentityHero 契约测试：资料域身份卡（/me 与 /users/[userId] 共用）的渲染契约——
// 身份信息/简介/位置/标签/指标齐全；在线状态只在 other 模式且数据已知时出现；
// 标题层级可注入（页面已有 PageHeader h1 时用 h2，整页唯一标题时用 h1）。
// snippet 类 props 在 jsdom 下不参与断言（见 frontend/AGENTS.md 组件测试约定）。
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

import IdentityHero from '../IdentityHero.svelte';

describe('IdentityHero', () => {
	it('渲染昵称/handle/简介/位置/标签与指标行', () => {
		render(IdentityHero, {
			props: {
				mode: 'self',
				name: '鹿鸣',
				handle: '@user_lumos',
				bio: '记录城市光影',
				location: '云龙湖',
				tags: ['摄影', '户外'],
				metrics: [
					{ label: '瞬间', value: 4 },
					{ label: '关注者', value: 5 },
					{ label: '群组', value: 2 }
				]
			}
		});
		expect(screen.getByRole('heading', { level: 2, name: '鹿鸣' })).toBeTruthy();
		expect(screen.getByText('@user_lumos')).toBeTruthy();
		expect(screen.getByText('记录城市光影')).toBeTruthy();
		expect(screen.getByText('云龙湖')).toBeTruthy();
		expect(screen.getByText('#摄影')).toBeTruthy();
		expect(screen.getByText('瞬间')).toBeTruthy();
		expect(screen.getByText('4')).toBeTruthy();
	});

	it('other 模式展示在线状态；self 模式不展示', () => {
		const { unmount } = render(IdentityHero, {
			props: { mode: 'other', name: '阿禾', online: true }
		});
		expect(screen.getByText('在线')).toBeTruthy();
		unmount();

		render(IdentityHero, { props: { mode: 'self', name: '鹿鸣', online: true } });
		expect(screen.queryByText('在线')).toBeNull();
	});

	it('other 模式在线状态未知（online 未传）时不假装在线', () => {
		render(IdentityHero, { props: { mode: 'other', name: '阿禾' } });
		expect(screen.queryByText('在线')).toBeNull();
		expect(screen.queryByText('离线')).toBeNull();
	});

	it('titleTag=h1 时昵称渲染为一级标题', () => {
		render(IdentityHero, { props: { mode: 'self', name: '鹿鸣', titleTag: 'h1' } });
		expect(screen.getByRole('heading', { level: 1, name: '鹿鸣' })).toBeTruthy();
	});
});
