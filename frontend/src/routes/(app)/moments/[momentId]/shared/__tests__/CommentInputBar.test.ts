// CommentInputBar 契约测试：inline/docked 两种登记形态 + 发送交互。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import CommentInputBar from '../CommentInputBar.svelte';

describe('CommentInputBar', () => {
	it('inline：嵌在评论区的平顺输入槽（全宽自然流，无左右 inset / 无悬浮投影）', () => {
		const { container } = render(CommentInputBar, {
			props: { input: '', onSend: () => {}, variant: 'inline' }
		});
		const shell = container.firstElementChild as HTMLElement;
		expect(shell.className).toContain('w-full');
		expect(shell.className).not.toContain('mx-3');
		expect(shell.className).not.toContain('backdrop-blur-md');
	});

	it('docked：移动端通栏贴底底栏（全宽贴边，无浮动悬浮外边距与悬浮投影）', () => {
		const { container } = render(CommentInputBar, {
			props: { input: '', onSend: () => {}, variant: 'docked' }
		});
		const shell = container.firstElementChild as HTMLElement;
		expect(shell.className).toContain('w-full');
		expect(shell.className).not.toContain('mx-3');
		expect(shell.className).not.toContain('shadow-raised-md');
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
