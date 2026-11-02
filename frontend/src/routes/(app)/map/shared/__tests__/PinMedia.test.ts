// PinMedia 契约测试：与全站列表媒体同语言——用户/群组走 Avatar、瞬间走圆角缩略图、
// 无图/聚合桶给中性无底色图标；不再有类型色砖底与 marker 光晕。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import PinMedia from '../PinMedia.svelte';

describe('PinMedia', () => {
	it('用户图钉渲染 Avatar 头像与在线角标', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'user', lat: 30.5, lng: 117.5, avatar: 'a.jpg', name: '鹿鸣', online: true } as never }
		});
		expect(container.querySelector('img')?.getAttribute('alt')).toBe('鹿鸣');
		// 角标统一走 OnlineDot（data-online-dot 稳定锚点）
		expect(container.querySelector('[data-online-dot]')).not.toBeNull();
	});

	it('群组图钉使用圆角 Avatar（无类型砖底色/光晕）', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'group', lat: 30.5, lng: 117.5, name: '测试群' } as never }
		});
		const box = container.firstElementChild as HTMLElement;
		expect(box.style.background).toBe('');
		expect(box.style.boxShadow).toBe('');
		// 群组头像为圆角方形（Avatar shape="rounded"），不是圆形/色砖
		expect(box.style.borderRadius).not.toBe('9999px');
		// 无群头像时回退群名首字母
		expect(container.textContent).toContain('测');
	});

	it('动态图钉渲染圆角缩略图', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'moment', lat: 30.5, lng: 117.5, image: 'm.jpg', title: '湖畔' } as never }
		});
		const img = container.querySelector('img');
		expect(img?.getAttribute('alt')).toBe('湖畔');
		// 缩略图容器为圆角媒体块（不是类型砖）
		expect(img?.closest('span')?.className).toContain('rounded-sm');
	});

	it('无图动态渲染中性无底色图标', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'moment', lat: 30.5, lng: 117.5, title: '无图动态' } as never }
		});
		const box = container.firstElementChild as HTMLElement;
		expect(box.style.background).toBe('');
		expect(box.style.boxShadow).toBe('');
		expect(container.querySelector('svg')).not.toBeNull();
	});

	it('聚合桶渲染无底色类型图标（不是头像砖）', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'heat:1', type: 'group', lat: 30.5, lng: 117.5, name: '6 个群组', isAggregate: true } as never }
		});
		expect(container.querySelector('svg')).not.toBeNull();
		expect(container.querySelector('img')).toBeNull();
	});

	it('选中态无底色图标换 on-accent 档（accent 行底对比度）', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'moment', lat: 30.5, lng: 117.5 } as never, selected: true }
		});
		const svg = container.querySelector('svg') as SVGElement;
		expect(svg.getAttribute('style')).toContain('--cst-text-on-accent');
	});
});
