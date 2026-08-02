import { describe, expect, it } from 'vitest';
import { requireNumericId, requireProfileSection } from './route-guards';

function expectHttp404(fn: () => unknown) {
	try {
		fn();
		expect.unreachable('应当抛出 404');
	} catch (e) {
		expect((e as { status?: number }).status).toBe(404);
	}
}

describe('requireNumericId', () => {
	it('接受纯数字 ID', () => {
		expect(requireNumericId('1')).toBe('1');
		expect(requireNumericId('123456789012345678')).toBe('123456789012345678');
	});

	it('拒绝非数字、负数、小数与空串', () => {
		expectHttp404(() => requireNumericId('abc'));
		expectHttp404(() => requireNumericId('12a'));
		expectHttp404(() => requireNumericId('-1'));
		expectHttp404(() => requireNumericId('1.5'));
		expectHttp404(() => requireNumericId(''));
	});
});

describe('requireProfileSection', () => {
	it('接受五个合法分区', () => {
		for (const section of ['moments', 'groups', 'friends', 'following', 'fans']) {
			expect(requireProfileSection(section)).toBe(section);
		}
	});

	it('拒绝未知分区', () => {
		expectHttp404(() => requireProfileSection('whatever'));
		expectHttp404(() => requireProfileSection('relationships'));
		expectHttp404(() => requireProfileSection(''));
	});
});
