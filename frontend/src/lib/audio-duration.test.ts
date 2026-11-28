import { describe, expect, it } from 'vitest';
import { needsDurationProbe, probeAudioDuration } from './audio-duration';

function makeAudio(initial: number) {
	const listeners: Record<string, Array<() => void>> = {};
	const node = {
		currentTime: 0,
		duration: initial,
		addEventListener(type: string, handler: () => void) {
			(listeners[type] ??= []).push(handler);
		},
		removeEventListener(type: string, handler: () => void) {
			listeners[type] = (listeners[type] ?? []).filter((h) => h !== handler);
		},
		fire(type: string) {
			for (const h of listeners[type] ?? []) h();
		}
	} as unknown as HTMLAudioElement & { fire: (t: string) => void };
	return { node, listeners };
}

describe('needsDurationProbe', () => {
	it('NaN / Infinity / 0 需要探测', () => {
		expect(needsDurationProbe(Number.NaN)).toBe(true);
		expect(needsDurationProbe(Number.POSITIVE_INFINITY)).toBe(true);
		expect(needsDurationProbe(0)).toBe(true);
	});

	it('有限正值不再探测', () => {
		expect(needsDurationProbe(7.2)).toBe(false);
	});
});

describe('probeAudioDuration', () => {
	it('WebM 无时长：loadedmetadata 跳大时间，durationchange 拿到时长后回到 0', () => {
		const { node } = makeAudio(Number.POSITIVE_INFINITY);
		probeAudioDuration(node);
		(node as unknown as { fire: (t: string) => void }).fire('loadedmetadata');
		expect(node.currentTime).toBe(1e101);
		(node as unknown as { duration: number }).duration = 7.2;
		(node as unknown as { fire: (t: string) => void }).fire('durationchange');
		expect(node.currentTime).toBe(0);
	});

	it('已有有限时长时不做任何跳转', () => {
		const { node } = makeAudio(2.4);
		probeAudioDuration(node);
		(node as unknown as { fire: (t: string) => void }).fire('loadedmetadata');
		expect(node.currentTime).toBe(0);
	});

	it('destroy 后不再响应事件', () => {
		const { node, listeners } = makeAudio(Number.NaN);
		const { destroy } = probeAudioDuration(node);
		destroy();
		expect((listeners['loadedmetadata'] ?? []).length).toBe(0);
		expect((listeners['durationchange'] ?? []).length).toBe(0);
	});
});
