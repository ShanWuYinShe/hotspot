import { describe, expect, it, vi, beforeEach } from 'vitest';
import { goto } from '$app/navigation';
import { smartBack } from './smart-back';
import { ROUTES } from '@/lib/routes';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

// smartBack 契约：history.length > 1 → history.back()；深链直达
//（length === 1，无处可退）→ goto fallback。
// 刻意不用 document.referrer：SPA 内导航不更新它，用它会把站内流转
//（如列表进详情再返回）误判成深链送去首页。
describe('smartBack', () => {
	beforeEach(() => {
		vi.mocked(goto).mockClear();
	});

	it('有历史可退时 history.back()（不看 referrer）', () => {
		Object.defineProperty(document, 'referrer', { value: '', configurable: true });
		Object.defineProperty(window, 'history', { value: { back: vi.fn(), length: 2 }, configurable: true });
		smartBack();
		expect(window.history.back).toHaveBeenCalledTimes(1);
		expect(goto).not.toHaveBeenCalled();
	});

	it('深链直达（history.length<=1）时 goto 首页', () => {
		Object.defineProperty(window, 'history', { value: { back: vi.fn(), length: 1 }, configurable: true });
		smartBack();
		expect(window.history.back).not.toHaveBeenCalled();
		expect(goto).toHaveBeenCalledWith(ROUTES.home);
	});

	it('支持指定 fallback', () => {
		Object.defineProperty(window, 'history', { value: { back: vi.fn(), length: 1 }, configurable: true });
		smartBack('/nearby/moments');
		expect(goto).toHaveBeenCalledWith('/nearby/moments');
	});
});
