// EmptyState 契约测试：标题/描述渲染、bare 模式与图标容器尺寸、spotlight 装饰层。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import EmptyState from '../EmptyState.svelte';

describe('EmptyState', () => {
	it('渲染标题与描述', () => {
		render(EmptyState, { props: { title: '暂无消息', description: '去打个招呼吧' } });
		expect(screen.getByText('暂无消息')).toBeTruthy();
		expect(screen.getByText('去打个招呼吧')).toBeTruthy();
	});

	it('未传 icon 时也渲染默认图标圆形容器', () => {
		const { container } = render(EmptyState, { props: { title: '暂无内容' } });
		expect(container.querySelector('.rounded-full')).not.toBeNull();
		expect(container.querySelector('.lucide-inbox')).not.toBeNull();
	});

	it('bare 模式无图标圆形容器', () => {
		const { container } = render(EmptyState, {
			props: { title: '空', bare: true, iconSize: 72 }
		});
		expect(container.querySelector('.rounded-full')).toBeNull();
	});

	it('spotlight 装饰层对读屏与指针隐藏，且 bare 内容契约不变', () => {
		const { container } = render(EmptyState, { props: { title: '空', bare: true, spotlight: true } });
		const layer = container.querySelector('[data-empty-spotlight]') as HTMLElement;
		expect(layer).not.toBeNull();
		expect(layer.getAttribute('aria-hidden')).toBe('true');
		expect(layer.className).toContain('empty-spotlight');
		expect(container.querySelector('.empty-spotlight__ring--outer')).not.toBeNull();
		expect(screen.getByText('空')).toBeTruthy();
	});

	it('默认不渲染 spotlight 装饰层', () => {
		const { container } = render(EmptyState, { props: { title: '空' } });
		expect(container.querySelector('[data-empty-spotlight]')).toBeNull();
	});
});
