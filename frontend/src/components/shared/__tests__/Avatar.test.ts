// Avatar 契约测试：图片渲染、失败回退占位、尺寸应用。
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Avatar from '../Avatar.svelte';

describe('Avatar', () => {
	it('有 src 时渲染 img 并带 alt', () => {
		render(Avatar, { props: { src: 'https://example.com/a.jpg', name: '鹿鸣' } });
		const img = screen.getByAltText('鹿鸣') as HTMLImageElement;
		expect(img).toBeTruthy();
	});

	it('src 为空时渲染首字母占位，不渲染 img', () => {
		const { container } = render(Avatar, { props: { name: '鹿鸣' } });
		expect(container.querySelector('img')).toBeNull();
		expect(screen.getByText('鹿')).toBeTruthy();
	});

	it('图片加载失败后回退为首字母占位', async () => {
		const { container } = render(Avatar, { props: { src: 'https://example.com/b.jpg', name: '清野' } });
		const img = container.querySelector('img') as Element;
		expect(img).not.toBeNull();
		await fireEvent.error(img);
		expect(container.querySelector('img')).toBeNull();
		expect(screen.getByText('清')).toBeTruthy();
	});

	it('size 应用到容器尺寸', () => {
		const { container } = render(Avatar, { props: { name: '测试', size: 48 } });
		const wrapper = container.firstElementChild as HTMLElement;
		expect(wrapper.style.width).toBe('48px');
		expect(wrapper.style.height).toBe('48px');
	});
});
