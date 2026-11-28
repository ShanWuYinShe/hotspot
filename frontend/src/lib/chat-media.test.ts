import { describe, expect, it } from 'vitest';
import { chatMediaKindOfFile, chatUploadPlanOfFile, formatFileSize, hasChatMedia, mapChatMediaFields, normalizeChatAudioMime } from './chat-media';

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

describe('chatMediaKindOfFile', () => {
	it('按 MIME 分流图片/视频', () => {
		expect(chatMediaKindOfFile({ type: 'image/png' })).toBe('image');
		expect(chatMediaKindOfFile({ type: 'video/mp4' })).toBe('video');
	});

	it('其它类型返回 null', () => {
		expect(chatMediaKindOfFile({ type: 'application/pdf' })).toBeNull();
		expect(chatMediaKindOfFile({ type: '' })).toBeNull();
	});
});

describe('chatUploadPlanOfFile', () => {
	it('图片/视频走 chat_image/chat_video，其余走 chat_file', () => {
		expect(chatUploadPlanOfFile({ type: 'image/png' })).toEqual({ kind: 'image', category: 'chat_image', maxBytes: 10 * 1024 * 1024 });
		expect(chatUploadPlanOfFile({ type: 'video/mp4' })).toEqual({ kind: 'video', category: 'chat_video', maxBytes: 100 * 1024 * 1024 });
		expect(chatUploadPlanOfFile({ type: 'application/pdf' })).toEqual({ kind: 'file', category: 'chat_file', maxBytes: 100 * 1024 * 1024 });
		expect(chatUploadPlanOfFile({ type: '' })).toEqual({ kind: 'file', category: 'chat_file', maxBytes: 100 * 1024 * 1024 });
	});
});

describe('formatFileSize', () => {
	it('B/KB/MB 三档', () => {
		expect(formatFileSize(900)).toBe('900 B');
		expect(formatFileSize(3421)).toBe('3.3 KB');
		expect(formatFileSize(2.4 * 1024 * 1024)).toBe('2.4 MB');
	});

	it('空值/0 返回空串', () => {
		expect(formatFileSize(0)).toBe('');
		expect(formatFileSize(null)).toBe('');
		expect(formatFileSize(undefined)).toBe('');
	});
});

describe('chatUploadPlanOfFile 音频档', () => {
	it('audio/* → chat_audio / 20MB', () => {
		expect(chatUploadPlanOfFile({ type: 'audio/mpeg' })).toEqual({ kind: 'audio', category: 'chat_audio', maxBytes: 20 * 1024 * 1024 });
	});
});

describe('normalizeChatAudioMime', () => {
	it('契约内 MIME 原样返回', () => {
		expect(normalizeChatAudioMime({ name: 'a.mp3', type: 'audio/mpeg' })).toBe('audio/mpeg');
	});

	it('m4a 报 audio/x-m4a 时归一为 audio/aac', () => {
		expect(normalizeChatAudioMime({ name: 'voice.m4a', type: 'audio/x-m4a' })).toBe('audio/aac');
		expect(normalizeChatAudioMime({ name: 'voice.m4a', type: '' })).toBe('audio/aac');
	});

	it('不支持的扩展返回 null', () => {
		expect(normalizeChatAudioMime({ name: 'a.wav', type: 'audio/wav' })).toBeNull();
	});
});

describe('audio/webm（MediaRecorder 录制）', () => {
	it('webm 已在契约白名单内，MIME 原样保留', () => {
		expect(normalizeChatAudioMime({ name: 'voice.webm', type: 'audio/webm' })).toBe('audio/webm');
		expect(normalizeChatAudioMime({ name: 'voice.webm', type: '' })).toBe('audio/webm');
	});
});
