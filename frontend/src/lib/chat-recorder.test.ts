import { describe, expect, it } from 'vitest';
import { formatRecordDuration, pickRecordMimeType } from './chat-recorder';

describe('pickRecordMimeType', () => {
	it('优先 opus，其次裸 webm', () => {
		expect(pickRecordMimeType((m) => m === 'audio/webm;codecs=opus')).toBe('audio/webm;codecs=opus');
		expect(pickRecordMimeType((m) => m === 'audio/webm')).toBe('audio/webm');
	});

	it('都不支持返回 null（Safari 的 audio/mp4 不在契约内）', () => {
		expect(pickRecordMimeType(() => false)).toBeNull();
	});
});

describe('formatRecordDuration', () => {
	it('mm:ss 两位补零', () => {
		expect(formatRecordDuration(0)).toBe('00:00');
		expect(formatRecordDuration(7_400)).toBe('00:07');
		expect(formatRecordDuration(65_000)).toBe('01:05');
	});

	it('负值归零', () => {
		expect(formatRecordDuration(-1000)).toBe('00:00');
	});
});
