import { describe, expect, it } from 'vitest';
import { buildReplyPreview, mediaPlaceholder } from './chat-reply';

describe('buildReplyPreview', () => {
	it('文本引用保留发送者与截断正文', () => {
		expect(
			buildReplyPreview({ message_id: 'm1', content: '明天下午三点老地方见', type: 'text', sender: { nickname: '阿禾' } })
		).toEqual({ id: 'm1', senderName: '阿禾', preview: '明天下午三点老地方见' });
	});

	it('超长文本截断到 40 字并加省略号', () => {
		const long = '一'.repeat(50);
		const preview = buildReplyPreview({ message_id: 'm2', content: long, type: 'text' });
		expect(preview?.preview).toBe('一'.repeat(40) + '…');
	});

	it('媒体消息用类型占位', () => {
		expect(buildReplyPreview({ message_id: 'm3', content: '', type: 'image' })?.preview).toBe('[图片]');
		expect(buildReplyPreview({ message_id: 'm4', content: null, type: 'file' })?.preview).toBe('[文件]');
	});

	it('缺少 message_id 返回 null', () => {
		expect(buildReplyPreview(null)).toBeNull();
		expect(buildReplyPreview({ content: 'x' })).toBeNull();
	});
});

describe('mediaPlaceholder', () => {
	it('覆盖五种媒体类型与兜底', () => {
		expect(mediaPlaceholder('video')).toBe('[视频]');
		expect(mediaPlaceholder('audio')).toBe('[语音]');
		expect(mediaPlaceholder('share')).toBe('[分享]');
		expect(mediaPlaceholder('unknown')).toBe('[消息]');
	});
});
