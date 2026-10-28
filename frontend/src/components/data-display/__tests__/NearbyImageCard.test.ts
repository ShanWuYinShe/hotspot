// NearbyImageCard 契约测试：图片/占位渲染、点击与键盘触发。
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import NearbyImageCard from '../NearbyImageCard.svelte';

describe('NearbyImageCard', () => {
	it('有图片时渲染 img 并带 alt', () => {
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'https://example.com/a.jpg', imageAlt: '群组封面' }
		});
		expect(container.querySelector('img')?.getAttribute('alt')).toBe('群组封面');
	});

	it('无图片时渲染渐变占位层', () => {
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: '', placeholder: (() => null) as never }
		});
		expect(container.querySelector('.surface-accent-gradient')).not.toBeNull();
		expect(container.querySelector('img')).toBeNull();
	});

	it('装饰占位层不带 z-index（footer tray/徽章绘制在上）', () => {
		// 回归防线：占位层曾带 z-[1]，无图时整层盖住 footer 与徽章——
		// DOM/a11y 一切正常，视觉上卡片只剩占位字母（线上实测，见 GroupCarousel 无图群组卡）。
		// jsdom 不做层叠绘制，只能靠类串断言拦截：装饰层禁止任何 z 工具类。
		for (const props of [
			{ imageSrc: '', placeholder: (() => null) as never },
			{ imageSrc: 'https://example.com/a.jpg' }
		] as never[]) {
			const { container, unmount } = render(NearbyImageCard, { props });
		for (const layer of Array.from(container.querySelectorAll('.surface-accent-gradient'))) {
				expect(layer.className).not.toMatch(/(?:^|\s)z-(?:\[|\d)/);
			}
			unmount();
		}
	});

	it('图片盒显式 w/h 拉满(替换元素仅 inset 全拉锚不拉伸)', () => {
		// 回归防线:abs 定位的 img 在 width/height:auto 时按固有尺寸渲染(CSS2 §10.3.7),
		// inset-0 拉不开、object-cover 无从生效——mock 小图在卡片内露底/错裁(线上实测)。
		// jsdom 不做布局,按本文件既有惯例以类串断言拦截:纵横两变体的 img 都必须
		// 带显式高度/宽度档(h-full + w-full / w-2/5)。
		const { container, unmount } = render(NearbyImageCard, {
			props: { imageSrc: 'https://example.com/a.jpg' }
		});
		const cls = container.querySelector('img')?.className ?? '';
		expect(cls).toContain('h-full');
		expect(cls).toContain('w-full');
		unmount();

		const horizontal = render(NearbyImageCard, {
			props: { imageSrc: 'https://example.com/a.jpg', horizontal: true }
		});
		const hCls = horizontal.container.querySelector('img')?.className ?? '';
		expect(hCls).toContain('h-full');
		expect(hCls).toContain('w-2/5');
		horizontal.unmount();
	});

	it('加载中占位用限时脉冲类（media-placeholder--pulse），不再用 Tailwind animate-pulse', () => {
		// R17：旧 animate-pulse 2s/infinite 会在慢图/lazy 卡上常驻；改 3 次后静态。
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'https://example.com/a.jpg' }
		});
		const placeholder = container.querySelector('.media-placeholder--pulse');
		expect(placeholder).not.toBeNull();
		expect(placeholder?.className).not.toContain('animate-pulse');
		expect(placeholder?.getAttribute('aria-hidden')).toBe('true');
	});

	it('onClick 提供时由覆盖按钮承载点击（根不再持 button 语义）', async () => {
		const onClick = vi.fn();
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick }
		});
		const card = container.querySelector('article') as HTMLElement;
		expect(card.getAttribute('role')).toBeNull();
		const cover = card.querySelector('[data-cover]') as HTMLElement;
		expect(cover.tagName).toBe('BUTTON');
		await fireEvent.click(cover);
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('按钮抬升类必须排除覆盖按钮自身（:not([data-cover])）', () => {
		// 回归防线:jsdom 不应用 Tailwind 样式表,塌缩(0×0)只能靠类串断言拦截——
		// [&_button]:relative 与覆盖按钮 absolute 同 specificity 且 .relative 后生成,
		// 一旦漏排除,真实浏览器整卡点击失效(线上实测 P0)
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick: () => {} }
		});
		const root = container.querySelector('article') as HTMLElement;
		expect(root.className).toContain('[&_button:not([data-cover])]:relative');
		expect(root.className).toContain('[&_button:not([data-cover])]:z-20');
		expect(root.className).not.toMatch(/\[&_button\]:(?:relative|z-\d+)/);
		// 卡片根自成堆叠上下文：内部按钮保持可点，整卡滚动到 sticky header
		// 下方时不再平级竞胜（用户反馈：卡片按钮压 header）。
		expect(root.className.split(/\s+/)).toContain('isolate');
	});

	it('覆盖按钮为原生 button（键盘 Enter/Space 原生激活）', async () => {
		const onClick = vi.fn();
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick }
		});
		const cover = container.querySelector('[data-cover]') as HTMLElement;
		expect(cover.tagName).toBe('BUTTON');
		expect((cover as HTMLButtonElement).type).toBe('button');
		await fireEvent.click(cover);
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('按压反馈由统一 cardPress 驱动（卡本体/覆盖按钮按下 → data-pressed，松开复位）', async () => {
		const { container } = render(NearbyImageCard, { props: { imageSrc: 'x', onClick: () => {} } });
		const card = container.querySelector('article') as HTMLElement;
		// 覆盖按钮是卡片本体的点击面：按下它等于按下整卡（应缩放）
		const cover = card.querySelector('[data-cover]') as HTMLElement;
		await fireEvent.pointerDown(cover);
		expect(card.dataset.pressed).toBe('true');
		await fireEvent.pointerUp(card);
		expect(card.dataset.pressed).toBeUndefined();
	});
});