import { describe, expect, it } from 'vitest';
import {
	FALLBACK_COMMENT_LIMIT,
	FALLBACK_MOMENT_TEXT_LIMIT,
	clampMomentText,
	resolveCommentLimit,
	resolveMomentTextLimit
} from './publish-limits';

describe('moment text limit', () => {
	it('未配置时回退到后端默认 2000', () => {
		expect(resolveMomentTextLimit()).toBe(2000);
		expect(FALLBACK_MOMENT_TEXT_LIMIT).toBe(2000);
	});

	it('优先使用配置值', () => {
		expect(resolveMomentTextLimit(280)).toBe(280);
	});

	it('非法/空配置回退默认值', () => {
		expect(resolveMomentTextLimit(0)).toBe(2000);
		expect(resolveMomentTextLimit(null)).toBe(2000);
	});

	it('clampMomentText 按注入上限截断', () => {
		expect(clampMomentText('a'.repeat(300), 280).length).toBe(280);
		expect(clampMomentText('短文本', 280)).toBe('短文本');
	});
});

describe('comment limit', () => {
	it('未配置时回退 500', () => {
		expect(resolveCommentLimit()).toBe(500);
		expect(FALLBACK_COMMENT_LIMIT).toBe(500);
	});

	it('优先使用配置值并对非法值回退', () => {
		expect(resolveCommentLimit(200)).toBe(200);
		expect(resolveCommentLimit(0)).toBe(500);
	});
});
