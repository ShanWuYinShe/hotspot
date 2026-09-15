// feed-progress 契约测试：存取/清空与定位 key 网格化。
import { describe, it, expect, beforeEach } from 'vitest';
import {
	getMomentsProgress,
	setMomentsProgress,
	clearMomentsProgress,
	feedProgressKey
} from '../feed-progress.svelte';

describe('moments feed progress', () => {
	beforeEach(() => {
		clearMomentsProgress();
	});

	it('初始无进度', () => {
		expect(getMomentsProgress()).toBeNull();
	});

	it('存取往返', () => {
		setMomentsProgress({ savedAt: Date.now(), key: 'k', items: [], nextCursor: 'c', visibleCount: 20, tags: ['咖啡'] });
		const saved = getMomentsProgress();
		expect(saved?.nextCursor).toBe('c');
		expect(saved?.visibleCount).toBe(20);
		clearMomentsProgress();
		expect(getMomentsProgress()).toBeNull();
	});

	it('定位 key 网格化到 ~100m（抖动不换 key）', () => {
		expect(feedProgressKey(null)).toBe('');
		const a = feedProgressKey({ latitude: 35.15492, longitude: 112.68252 });
		const b = feedProgressKey({ latitude: 35.15493, longitude: 112.68253 });
		expect(a).toBe(b);
		expect(feedProgressKey({ latitude: 36.0, longitude: 112.0 })).not.toBe(a);
	});
});
