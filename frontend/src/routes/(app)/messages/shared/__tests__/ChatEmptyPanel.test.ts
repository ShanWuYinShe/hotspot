// ChatEmptyPanel 契约测试：只做引导，不重复左栏的会话/新会话入口，小面板不开 spotlight。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import ChatEmptyPanel from '../ChatEmptyPanel.svelte';

describe('ChatEmptyPanel', () => {
	it('渲染引导标题与描述', () => {
		render(ChatEmptyPanel);
		expect(screen.getByText('选择一个会话')).toBeTruthy();
		expect(screen.getByText('从左侧列表选择会话继续聊天。')).toBeTruthy();
	});

	it('不在空面板内重复会话列表或新会话按钮（由左栏唯一承载）', () => {
		render(ChatEmptyPanel);
		expect(screen.queryByRole('button')).toBeNull();
	});

	it('窄面板内不开 spotlight 装饰层（空态规范：小面板不开）', () => {
		const { container } = render(ChatEmptyPanel);
		expect(container.querySelector('[data-empty-spotlight]')).toBeNull();
	});
});
