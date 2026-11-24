// MessageBubble 契约测试：文本/时间渲染、方向形态与已读双勾。
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
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
	// R16 排版/对比契约
	it('正文 14px（text-md）+ 卡片/气泡行高档 leading-card；时间戳 11px（xs）', () => {
		render(MessageBubble, { props: { text: '周末见', sent: false, time: '10:24' } });
		expect(screen.getByText('周末见').className).toContain('text-md');
		expect(screen.getByText('周末见').className).toContain('leading-card');
		expect(screen.getByText('10:24').className).toContain('text-xs');
	});

	it('自己消息时间戳不叠 opacity（accent 实底上保持 ≥4.5 对比）', () => {
		render(MessageBubble, { props: { text: '周末见', sent: true, time: '10:24' } });
		const ts = screen.getByText('10:24');
		expect(ts.className).toContain('text-text-on-accent');
		expect(ts.className).not.toContain('opacity-soft');
		expect(ts.className).not.toContain('opacity-emphasis');
	});

	it('群聊发送者名 12px（text-sm）', () => {
		render(MessageBubble, { props: { text: '收到', sent: false, time: '10:25', senderName: '阿禾' } });
		expect(screen.getByText('阿禾').className).toContain('text-sm');
	});

	it('气泡圆角统一 rounded-md（不再四角不对称的碎片观感）', () => {
		const { container } = render(MessageBubble, { props: { text: '收到', sent: false, time: '10:25' } });
		const bubble = container.querySelector('.max-w-chat-bubble') as HTMLElement;
		expect(bubble.className).toContain('rounded-md');
		expect(bubble.className).not.toContain('rounded-[');
	});

	it('撤回消息走中性气泡底色（不在 accent 实底上渲染低对比斜体）', () => {
		const { container } = render(MessageBubble, {
			props: { text: '[消息已撤回]', sent: true, time: '10:24', recalled: true }
		});
		expect(container.querySelector('.bg-accent-default')).toBeNull();
		expect(screen.getByText('消息已撤回')).toBeTruthy();
	});
});

	// 图片消息：不套气泡、渲染成可键盘触达的「查看大图」按钮，点击透出 URL（#263 契约）
	it('图片消息渲染为「查看大图」按钮且不套气泡底色', () => {
		const { container } = render(MessageBubble, {
			props: { text: '', imageUrl: 'https://cdn.example.com/a.jpg', sent: false, time: '10:24' }
		});
		expect(container.querySelector('button[aria-label="查看大图"]')).not.toBeNull();
		expect(container.querySelector('img')?.getAttribute('src')).toBe('https://cdn.example.com/a.jpg');
		// 图片分支不套气泡内距/底色（外层只剩布局用 max-w-chat-bubble）
		expect(container.querySelector('.px-4')).toBeNull();
		expect(container.querySelector('.bg-accent-default')).toBeNull();
		expect(container.querySelector('.bg-surface-primary')).toBeNull();
	});

	it('点击图片按钮回调 onImageClick(url)', async () => {
		const clicked: string[] = [];
		const { container } = render(MessageBubble, {
			props: { text: '', imageUrl: 'https://cdn.example.com/a.jpg', sent: true, time: '10:24', onImageClick: (url: string) => clicked.push(url) }
		});
		await fireEvent.click(container.querySelector('button[aria-label="查看大图"]') as HTMLButtonElement);
		expect(clicked).toEqual(['https://cdn.example.com/a.jpg']);
	});
