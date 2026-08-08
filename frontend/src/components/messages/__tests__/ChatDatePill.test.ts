// ChatDatePill 契约测试：默认/自定义日期文案与胶囊 token 类。
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

	it('胶囊应用主题 token 类', () => {
		const { container } = render(ChatDatePill, { props: {} });
		const pill = container.querySelector('span') as HTMLElement;
		expect(pill.className).toContain('bg-surface-primary');
		expect(pill.className).toContain('text-text-secondary');
	});
});
