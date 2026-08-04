import { describe, expect, it } from 'vitest';
import { requireProfileSection } from './route-guards';

function expectHttp404(fn: () => unknown) {
	try {
		fn();
		expect.unreachable('应当抛出 404');
	} catch (e) {
		expect((e as { status?: number }).status).toBe(404);
	}
}

describe('requireProfileSection', () => {
	it('接受合法分区', () => {
		for (const section of ['moments', 'groups', 'friends', 'following', 'fans', 'relationships']) {
			expect(requireProfileSection(section)).toBe(section);
		}
	});

	it('拒绝未知分区', () => {
		expectHttp404(() => requireProfileSection('whatever'));
		expectHttp404(() => requireProfileSection(''));
	});
});
