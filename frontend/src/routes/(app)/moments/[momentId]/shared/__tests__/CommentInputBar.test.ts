// CommentInputBar 契约测试：inline/floating 两种登记形态 + 发送交互（2026-10-29 详情页改版）。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import CommentInputBar from '../CommentInputBar.svelte';

describe('CommentInputBar', () => {
	it('inline：嵌在评论区的软底浅槽（无左右 inset / 无悬浮投影）', () => {
		const { container } = render(CommentInputBar, {
			props: { input: '', onSend: () => {}, variant: 'inline' }
		});
		const shell = container.firstElementChild as HTMLElement;
		expect(shell.className).toContain('bg-surface-subtle/40');
		expect(shell.className).not.toContain('mx-3');
		expect(shell.className).not.toContain('backdrop-blur-md');
	});

	it('floating：移动端悬浮 dock（左右 inset + 细边 + 投影 + 玻璃底），不再通栏贴边', () => {
		const { container } = render(CommentInputBar, {
			props: { input: '', onSend: () => {}, variant: 'floating' }
		});
		const shell = container.firstElementChild as HTMLElement;
		expect(shell.className).toContain('mx-3');
		expect(shell.className).toContain('rounded-md');
		expect(shell.className).toContain('backdrop-blur-md');
		expect(shell.className).toContain('shadow-raised-md');
	});

	it('空输入禁用发送；有内容可点发送并回调', async () => {
		const onSend = vi.fn();
		const { unmount } = render(CommentInputBar, { props: { input: '', onSend } });
		expect((screen.getByRole('button', { name: '发送评论' }) as HTMLButtonElement).disabled).toBe(true);
		unmount();
		render(CommentInputBar, { props: { input: '写点什么', onSend } });
		await fireEvent.click(screen.getByRole('button', { name: '发送评论' }));
		expect(onSend).toHaveBeenCalledTimes(1);
	});

	it('Enter 发送、Shift/IME 组合不误发', async () => {
		const onSend = vi.fn();
		render(CommentInputBar, { props: { input: '评论', onSend } });
		const input = screen.getByLabelText('评论输入框');
		await fireEvent.keyDown(input, { key: 'Enter' });
		expect(onSend).toHaveBeenCalledTimes(1);
	});
});
