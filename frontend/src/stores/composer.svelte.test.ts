// Composer store 契约测试：草稿跨挂载存活、发布成功后清空。
import { describe, it, expect, beforeEach } from 'vitest';
import { draft, hasDraft, clearDraft } from './composer.svelte';

describe('composer draft', () => {
	beforeEach(() => {
		clearDraft();
	});

	it('初始无草稿', () => {
		expect(hasDraft()).toBe(false);
		expect(draft.text).toBe('');
		expect(draft.moods).toEqual([]);
		expect(draft.images).toEqual([]);
		expect(draft.audience).toBe('nearby');
	});

	it('写入草稿后 hasDraft 为 true（模拟弹层关闭重开：store 不随组件卸载）', () => {
		draft.text = '未发出去的内容';
		draft.moods = ['咖啡'];
		draft.audience = 'private';
		expect(hasDraft()).toBe(true);
		expect(draft.text).toBe('未发出去的内容');
		expect(draft.moods).toEqual(['咖啡']);
		expect(draft.audience).toBe('private');
	});

	it('空白文本不算草稿', () => {
		draft.text = '   ';
		expect(hasDraft()).toBe(false);
	});

	it('clearDraft 重置全部字段', () => {
		draft.text = 'x';
		draft.moods = ['夜色'];
		draft.images = [new File(['x'], 'x.png')];
		draft.audience = 'friends';
		clearDraft();
		expect(hasDraft()).toBe(false);
		expect(draft.audience).toBe('nearby');
	});
});
