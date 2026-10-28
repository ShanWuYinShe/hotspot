// ChatDatePill 契约测试：默认/自定义日期文案；R16 起为「中性分节标签」——
// 无底色/无投影/非胶囊，避免在聊天画布上被误读为可点按钮。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import ChatDatePill from '../ChatDatePill.svelte';

describe('ChatDatePill', () => {
	it('默认渲染「今天」', () => {
		render(ChatDatePill, { props: {} });
		expect(screen.getByText('今天')).toBeTruthy();
	});

	it('自定义 label 渲染', () => {
		render(ChatDatePill, { props: { label: '昨天' } });
		expect(screen.getByText('昨天')).toBeTruthy();
	});

	it('中性分节标签：11px 次级文字 + 居中，无底色/投影/胶囊外观（不写成按钮观感）', () => {
		const { container } = render(ChatDatePill, { props: {} });
		const root = container.querySelector('[data-chat-date-label]') as HTMLElement;
		expect(root.className).toContain('justify-center');
		const label = screen.getByText('今天');
		expect(label.className).toContain('text-xs');
		expect(label.className).toContain('text-text-secondary');
		expect(label.className).not.toContain('bg-surface-primary');
		expect(label.className).not.toContain('rounded-full');
		expect(label.className).not.toContain('shadow-xs');
	});
});
