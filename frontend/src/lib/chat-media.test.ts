import { describe, expect, it } from 'vitest';
import { hasChatMedia, mapChatMediaFields } from './chat-media';

describe('mapChatMediaFields', () => {
	it('图片 → imageUrl', () => {
		expect(mapChatMediaFields('image', { url: 'u1' })).toEqual({ imageUrl: 'u1' });
	});

	it('视频 → videoUrl（此前被忽略）', () => {
		expect(mapChatMediaFields('video', { url: 'u2' })).toEqual({ videoUrl: 'u2' });
	});

	it('语音 → audioUrl', () => {
		expect(mapChatMediaFields('audio', { url: 'u3' })).toEqual({ audioUrl: 'u3' });
	});

	it('文件 → fileUrl + 文件名兜底', () => {
		expect(mapChatMediaFields('file', { url: 'u4', file_name: '报告.pdf' })).toEqual({ fileUrl: 'u4', fileName: '报告.pdf' });
		expect(mapChatMediaFields('file', { url: 'u5' })).toEqual({ fileUrl: 'u5', fileName: '附件' });
	});

	it('撤回或无 url 时清空媒体字段', () => {
		expect(mapChatMediaFields('video', { url: 'u6' }, true)).toEqual({});
		expect(mapChatMediaFields('image', null)).toEqual({});
		expect(mapChatMediaFields('text', { url: 'u7' })).toEqual({});
	});
});

describe('hasChatMedia', () => {
	it('任一媒体字段存在即为媒体态', () => {
		expect(hasChatMedia({ imageUrl: 'a' })).toBe(true);
		expect(hasChatMedia({ videoUrl: 'b' })).toBe(true);
		expect(hasChatMedia({ audioUrl: 'c' })).toBe(true);
		expect(hasChatMedia({ fileUrl: 'd' })).toBe(true);
		expect(hasChatMedia({})).toBe(false);
	});
});
