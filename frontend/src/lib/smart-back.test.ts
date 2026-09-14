import { describe, expect, it, vi, beforeEach } from 'vitest';
import { goto } from '$app/navigation';
import { smartBack } from './smart-back';
import { ROUTES } from '@/lib/routes';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

// smartBack 契约：站内历史 → history.back()；深链直达（无站内历史）→ goto fallback。
describe('smartBack', () => {
	beforeEach(() => {
		vi.mocked(goto).mockClear();
	});

	it('站内历史存在时 history.back()', () => {
		Object.defineProperty(document, 'referrer', { value: `${window.location.origin}/nearby/moments`, configurable: true });
		Object.defineProperty(window, 'history', { value: { back: vi.fn(), length: 2 }, configurable: true });
		smartBack();
		expect(window.history.back).toHaveBeenCalledTimes(1);
		expect(goto).not.toHaveBeenCalled();
	});

	it('深链直达（history.length<=1）时 goto 首页', () => {
		Object.defineProperty(document, 'referrer', { value: '', configurable: true });
		Object.defineProperty(window, 'history', { value: { back: vi.fn(), length: 1 }, configurable: true });
		smartBack();
		expect(goto).toHaveBeenCalledWith(ROUTES.home);
	});

	it('站外 referrer 时 goto 指定 fallback', () => {
		Object.defineProperty(document, 'referrer', { value: 'https://evil.example/post', configurable: true });
		Object.defineProperty(window, 'history', { value: { back: vi.fn(), length: 5 }, configurable: true });
		smartBack('/nearby/moments');
		expect(goto).toHaveBeenCalledWith('/nearby/moments');
	});
});
