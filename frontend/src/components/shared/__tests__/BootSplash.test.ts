// BootSplash 契约测试：启动动画的结构完整性（状态语义 / SVG 图层 / logo 内嵌）。
// 动画本身由 CSS 承载（keyframes + prefers-reduced-motion 全局降级），jsdom 不
// 计算 CSS 动画，这里只锁结构与无障碍契约。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BootSplash from '../BootSplash.svelte';

describe('BootSplash', () => {
	it('以 role=status 暴露加载语义与中文可访问名', () => {
		const { container } = render(BootSplash);
		const status = container.querySelector('[role="status"]');
		expect(status).not.toBeNull();
		expect(status?.getAttribute('aria-label')).toBe('加载中');
	});

	it('SVG 含三层动画结构：外环轨道+旋转弧、反向虚线环、logo 呼吸', () => {
		const { container } = render(BootSplash);
		expect(container.querySelector('.orbit')).not.toBeNull();
		expect(container.querySelector('.orbit-rev')).not.toBeNull();
		expect(container.querySelector('.logo-breath')).not.toBeNull();
		// 弧线渐变引用存在
		expect(container.querySelector('linearGradient')).not.toBeNull();
		expect(container.querySelector('image[href]')).not.toBeNull();
	});
});
