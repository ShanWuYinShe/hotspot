// PinMedia 契约测试：用户头像/动态缩略/marker 与选中底。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import PinMedia from '../PinMedia.svelte';

describe('PinMedia', () => {
	it('用户图钉渲染头像与在线角标', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'user', lat: 30.5, lng: 117.5, avatar: 'a.jpg', name: '鹿鸣', online: true } as never }
		});
		expect(container.querySelector('img')?.getAttribute('alt')).toBe('鹿鸣');
		expect(container.querySelector('.bg-status-online')).not.toBeNull();
	});

	it('动态图钉渲染图片缩略', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'moment', lat: 30.5, lng: 117.5, image: 'm.jpg', title: '湖畔' } as never }
		});
		expect(container.querySelector('img')?.getAttribute('alt')).toBe('湖畔');
	});

	it('群组图钉渲染 marker SVG 与淡底', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'group', lat: 30.5, lng: 117.5 } as never }
		});
		expect(container.querySelector('svg')).not.toBeNull();
		expect((container.firstElementChild as HTMLElement).style.background).toContain('var(--cst-marker-');
	});

	it('selected 时 marker 底换实色', () => {
		const { container } = render(PinMedia, {
			props: { pin: { id: 'p1', type: 'group', lat: 30.5, lng: 117.5 } as never, selected: true }
		});
		expect((container.firstElementChild as HTMLElement).style.background).toContain('var(--cst-surface-primary)');
	});
});
