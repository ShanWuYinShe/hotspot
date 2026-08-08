// SettingRow 契约测试：标题/描述渲染与图标容器。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import SettingRow from '../SettingRow.svelte';

describe('SettingRow', () => {
	it('渲染标题与描述', () => {
		render(SettingRow, {
			props: { title: '外观', description: '主题与深色模式', icon: (() => null) as never }
		});
		expect(screen.getByText('外观')).toBeTruthy();
		expect(screen.getByText('主题与深色模式')).toBeTruthy();
	});

	it('无 description 时不渲染副标题', () => {
		render(SettingRow, { props: { title: '账号信息', icon: (() => null) as never } });
		expect(screen.getByText('账号信息')).toBeTruthy();
		const container = document.body;
		expect(container.querySelectorAll('p').length).toBe(1);
	});

	it('渲染左侧 icon-badge 圆形容器', () => {
		const { container } = render(SettingRow, {
			props: { title: '通知', icon: (() => null) as never }
		});
		expect(container.querySelector('.icon-badge')).not.toBeNull();
	});
});
