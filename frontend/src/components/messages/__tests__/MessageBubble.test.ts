// MessageBubble 契约测试：文本/时间渲染、方向形态与已读双勾。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import MessageBubble from '../MessageBubble.svelte';

describe('MessageBubble', () => {
	it('渲染文本与时间', () => {
		render(MessageBubble, { props: { text: '周末见', sent: true, time: '10:24' } });
		expect(screen.getByText('周末见')).toBeTruthy();
		expect(screen.getByText('10:24')).toBeTruthy();
	});

	it('sent=true 使用发送方形态（右对齐 + accent，昵称列布局）', () => {
		const { container } = render(MessageBubble, { props: { text: '周末见', sent: true, time: '10:24' } });
		const wrapper = container.firstElementChild as HTMLElement;
		expect(wrapper.className).toContain('items-end');
		expect(container.querySelector('.bg-accent-default')).not.toBeNull();
	});

	it('sent=false 使用接收方形态（左对齐 + 表面，昵称列布局）', () => {
		const { container } = render(MessageBubble, { props: { text: '收到', sent: false, time: '10:25' } });
		const wrapper = container.firstElementChild as HTMLElement;
		expect(wrapper.className).toContain('items-start');
		expect(container.querySelector('.bg-surface-primary')).not.toBeNull();
	});

	it('showCheck 时渲染已读双勾', () => {
		const { container } = render(MessageBubble, {
			props: { text: '周末见', sent: true, time: '10:24', showCheck: true }
		});
		expect(container.querySelector('svg')).not.toBeNull();
	});
});
