// people-progress 契约测试：存取/清空与定位 key 网格化。
import { describe, it, expect, beforeEach } from 'vitest';
import {
	getPeopleProgress,
	setPeopleProgress,
	clearPeopleProgress,
	peopleProgressKey
} from '../people-progress.svelte';

describe('people list progress', () => {
	beforeEach(() => {
		clearPeopleProgress();
	});

	it('初始无进度', () => {
		expect(getPeopleProgress()).toBeNull();
	});

	it('存取往返', () => {
		setPeopleProgress({ key: 'k', users: [], visibleCount: 12 });
		expect(getPeopleProgress()?.visibleCount).toBe(12);
		clearPeopleProgress();
		expect(getPeopleProgress()).toBeNull();
	});

	it('定位 key 网格化到 ~100m（抖动不换 key）', () => {
		expect(peopleProgressKey(null)).toBe('');
		const a = peopleProgressKey({ latitude: 35.15492, longitude: 112.68252 });
		const b = peopleProgressKey({ latitude: 35.15493, longitude: 112.68253 });
		expect(a).toBe(b);
	});
});
